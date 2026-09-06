Sen kıdemli bir sistem mimarı ve Go geliştiricisisin.

Üzerinde çalıştığımız proje:

# ORH (Open Runtime for AI Orchestration)

ORH, yapay zeka orchestration mimarilerini tanımlamak, paylaşmak ve çalıştırmak için açık kaynak bir runtime'dır.

Faz 1 tamamlandı.

Mevcut sistem:

* `.orh` dosyası okunabiliyor.
* YAML parser mevcut.
* Validation mevcut.
* AgentComponent mevcut.
* Ollama provider mevcut.
* Tek component çalıştırılabiliyor.
* CLI mevcut.

Şimdi Faz 2 geliştirilecek.

---

# Faz 2 Ana Hedef

ORH artık tek agent çalıştıran bir araç değil, bir **execution graph runtime** haline gelmelidir.

Kullanıcı bir `.orh` dosyasında:

* birden fazla component tanımlayabilmeli,
* component'ler arasında bağlantı kurabilmeli,
* runtime bu graph'ı çalıştırabilmeli.

Örneğin:

Input:

```
        ┌───────┐
        │ AgentA│
Input ──►       │
        └───┬───┘
            │
            ▼
        ┌───────┐
        │ AgentB│
        └───────┘
            │
            ▼
         Output
```

çalışmalı.

---

# Temel Mimari Kural

ORH herhangi bir orchestration türünü özel olarak implement etmemelidir.

Yani:

YANLIŞ:

```
SequentialEngine
DebateEngine
SupervisorEngine
```

oluşturma.

DOĞRU:

Generic graph runtime oluştur.

Sequential veya debate gibi yapılar sadece graph tanımı olarak oluşmalıdır.

---

# Faz 2 Yeni Kavramları

Runtime artık aşağıdaki kavramları desteklemeli:

## Component

Bir execution birimidir.

Örnek:

* Agent
* Future tool
* Future custom node

Interface:

```go
type Component interface {

    Handle(
        ctx context.Context,
        event Event,
    ) ([]Command,error)

}
```

---

## Event

Component'ler arası veri taşıma mekanizması.

Örnek:

```go
type Event struct {

    Source string

    Target string

    Port string

    Payload any

}
```

---

## Connection

Component bağlantısı.

Örnek:

```
agent1.output
       |
       |
agent2.input
```

---

## Graph

Architecture'ın runtime modeli.

Örnek:

```go
type Graph struct {

    Components map[string]ComponentDefinition

    Connections []Connection

}
```

---

# Yeni Runtime Tasarımı

Runtime event-driven çalışmalı.

Akış:

```
Input Event

      |
      v

Scheduler

      |
      v

Component Handle()

      |
      v

Commands

      |
      v

New Events

```

---

# Command sistemi ekle

Component direkt runtime değiştirmemeli.

Command üretmeli.

Örnek:

```go
type Command interface{}
```

İlk commandler:

```
EmitEvent
SetOutput
```

Gelecekte:

```
SpawnComponent
CallTool
UpdateState
```

eklenebilecek şekilde tasarla.

---

# Scheduler oluştur

Basit bir scheduler yeterli.

Görevi:

* event queue yönetmek
* doğru component'i bulmak
* component'i çalıştırmak
* yeni eventleri kuyruğa koymak

Örnek:

```
Event Queue

[
 agent1.input
]


Scheduler

      |
      v


Agent1


      |
      v


agent2.input

```

---

# .orh Formatını Genişlet

Faz 1:

Tek component:

```yaml
components:

 assistant:

   type: agent

```

Faz 2:

Birden fazla component:

```yaml
apiVersion: orh/v1

name: simple-chain


components:


 researcher:

   type: agent

   model: qwen


 writer:

   type: agent

   model: qwen



connections:


 - from: input

   to: researcher.input



 - from: researcher.output

   to: writer.input



 - from: writer.output

   to: output

```

---

# Yeni Component Lifecycle

Component yaşam döngüsünü düşün:

```
Create

 ↓

Initialize

 ↓

Handle Event

 ↓

Produce Commands

 ↓

Destroy

```

İlk fazda minimal implementasyon yeterli.

---

# Parallel Execution Hazırlığı

Şimdiden runtime:

```
Input

 ├── AgentA

 ├── AgentB

 └── AgentC


        ↓


     Aggregator

```

destekleyebilecek yapıda olmalı.

Şimdilik özel aggregator yazmana gerek yok.

Sadece scheduler aynı anda bağımsız eventleri işleyebilecek şekilde tasarla.

---

# Graph Validation

Yeni validator kontrolleri ekle:

Kontrol:

* component mevcut mu?
* connection source var mı?
* connection target var mı?
* port isimleri geçerli mi?
* cycle destekleniyor mu?

Not:

Graph cycle desteği ileride lazım olacak.

DAG zorunluluğu koyma.

---

# CLI Güncellemeleri

Yeni komutlar:

## Graph gösterme

```
orh inspect architecture.orh
```

Örnek:

```
Architecture: research-chain


Components:

✓ researcher
✓ writer


Connections:

input
 |
researcher
 |
writer
 |
output

```

---

## Runtime debug

```
orh run architecture.orh --debug
```

Çıktı:

```
Event:
input


Executing:
researcher


Output:
researcher.output


Executing:
writer

Completed

```

---

# Test Senaryoları

Aşağıdaki örnekler çalışmalı:

## 1. Sequential

```
Input

 |

Agent A

 |

Agent B

 |

Output

```

---

## 2. Fan-out

```
        Agent A
       /
Input
       \
        Agent B

```

Şimdilik sonuçları birleştirmek zorunda değil.

---

## 3. Chain of 3 Agents

```
A -> B -> C

```

---

# Kod Organizasyonu

Mevcut yapıyı genişlet:

```
internal/

 runtime/

    scheduler/

    events/

    executor/


 graph/

    graph.go

    connection.go


 components/

    component.go

    agent.go


 commands/

    command.go

```

---

# Faz 2 Başarı Kriteri

Aşağıdaki çalışmalı:

Dosya:

```
research.orh
```

İçerik:

```
Agent1 -> Agent2 -> Agent3
```

Komut:

```
orh run research.orh
```

Runtime:

1. Graph parse eder.
2. Component instance oluşturur.
3. Event queue başlatır.
4. Input event gönderir.
5. Agent1 çalışır.
6. Çıktı Agent2'ye gider.
7. Agent3 çalışır.
8. Output döner.

---

# Önemli Tasarım Sorusu

Kod yazarken sürekli şu prensibi koru:

"ORH bir orchestration pattern framework değildir.

ORH generic AI execution runtime'dır."

Faz 2 sonunda elimizde:

* çoklu agent çalıştırabilen,
* graph tabanlı,
* event-driven,
* genişletilebilir

bir ORH çekirdeği olmalı.

Bundan sonraki Faz 3:

* GitHub package resolver
* `orh pull`
* dependency sistemi
* architecture paylaşımı

olacaktır.




