


Sen kıdemli bir sistem mimarı ve Go backend geliştiricisisin.

Bir açık kaynak proje geliştiriyoruz: **ORH (Open Runtime for AI Orchestration)**.

Projenin amacı, yapay zeka orchestration mimarilerini model bağımsız, paylaşılabilir ve çalıştırılabilir hale getiren bir runtime oluşturmaktır.

ORH uzun vadede AI sistemleri için bir çalışma katmanı olacaktır ancak ilk fazda kapsamı bilinçli olarak küçük tutulacaktır.

## İlk Faz Hedefi

İlk fazın amacı:

Bir kullanıcının lokalindeki `.orh` dosyasını okuyup, tanımlanan agent graph'ını çalıştırabilmesi ve Ollama üzerinden bir model çağırabilmesidir.

İlk versiyonda sadece:

* `.orh` architecture formatı
* parser
* validator
* execution graph
* agent component
* Ollama model provider
* CLI

geliştirilecektir.

Şimdilik:

* toolbox (`.tb`)
* custom node (`.node`)
* registry
* web platformu
* plugin sistemi
* distributed execution

yapılmayacaktır.

Ancak mimari ileride bu özelliklerin eklenmesine izin verecek şekilde tasarlanmalıdır.

---

# Temel Tasarım Prensipleri

## 1. ORH orchestration pattern dayatmaz

Runtime içinde:

* DebateEngine
* SupervisorEngine
* SequentialEngine
* SwarmEngine

gibi özel engine'ler oluşturma.

ORH bunların hiçbirini bilmemelidir.

Bunun yerine generic execution modeli kur:

* Component
* Event
* State
* Connection
* Execution

gelecekte farklı architecture'ların bu primitive'ler üzerinden oluşturulmasını sağla.

---

## 2. Model bağımsızlığı

Architecture belirli bir modele bağlı olmamalıdır.

Yanlış:

model: llama3

Doğru yaklaşım:

model slot veya provider abstraction kullanmak.

İlk fazda sadece Ollama desteklenecek ama ileride:

* OpenAI compatible API
* vLLM
* Anthropic
* diğer providerlar

eklenebilecek şekilde interface tasarla.

---

## 3. Runtime yaklaşımı

Execution sistemi event-driven düşünülmelidir.

Temel fikir:

Input event gelir.

Component çalışır.

Output event üretir.

Output başka component'e aktarılır.

İlk fazda basit graph execution yeterlidir.

---

# Teknoloji Seçimi

Backend dili:

Go

CLI:

Cobra veya benzeri Go CLI framework kullan.

Config parsing:

YAML destekle.

Test:

Go testing framework kullan.

Kod:

production kalitesinde,
modüler,
temiz,
genişletilebilir yazılmalı.

---

# Proje Yapısı

Aşağıdaki gibi bir yapı öner:

orh/

cmd/
orh/

internal/

```
spec/
    .orh format modelleri

parser/
    yaml parsing

validator/
    syntax ve semantic validation

graph/
    execution graph

runtime/
    executor
    scheduler
    events

providers/
    ollama/

cli/
```

examples/

tests/

---

# İlk Faz CLI Komutları

Aşağıdaki komutları implement et:

## Proje oluşturma

orh init

Örnek:

my-agent/

main.orh

## Validation

orh validate main.orh

Beklenen:

✓ syntax valid
✓ components valid
✓ connections valid

## Çalıştırma

orh run main.orh

Opsiyon:

orh run main.orh --model ollama:qwen3

---

# İlk .orh Formatı

Basit bir başlangıç formatı tasarla.

Örnek:

```yaml
apiVersion: orh/v1

name: hello-agent

models:

  main:
    provider: ollama


components:

  assistant:

    type: agent

    model: main

    prompt: |
      Answer the user question.


connections:

  - from: input
    to: assistant.input


  - from: assistant.output
    to: output
```

Bu format ileride:

* toolbox
* custom node
* sub orchestration

destekleyebilecek şekilde tasarlanmalı.

---

# Agent Component

İlk component:

AgentComponent

Sorumlulukları:

* input almak
* prompt oluşturmak
* model çağırmak
* output üretmek

Interface öner:

```go
type Component interface {

    Handle(
        ctx context.Context,
        event Event,
    ) ([]Command,error)

}
```

Basit başlayabilirsin ancak ileride:

* spawn
* state
* tool call

eklenmesine uygun olsun.

---

# Ollama Provider

Bir abstraction oluştur:

```go
type ModelProvider interface {

    Generate(
        ctx context.Context,
        request Request,
    ) Response

}
```

İlk implementasyon:

OllamaProvider

---

# İlk Faz Başarı Kriteri

Aşağıdaki senaryo çalışmalı:

Kullanıcı:

1. main.orh oluşturur.

2. Şunu çalıştırır:

orh run main.orh

3. ORH:

* dosyayı okur
* validate eder
* graph oluşturur
* agent component oluşturur
* Ollama modelini çağırır
* sonucu döndürür

---

# Geliştirme Stratejisi

Önce:

1. Repository oluştur.
2. Go module oluştur.
3. CLI skeleton oluştur.
4. `.orh` parser yaz.
5. Validator yaz.
6. Graph modeli oluştur.
7. Runtime executor yaz.
8. Ollama adapter yaz.
9. End-to-end demo yap.

Kod yazarken her kararda şu soruyu sor:

"Bu karar ORH'nin gelecekte farklı architecture, toolbox ve custom component desteklemesini engeller mi?"

Eğer engelliyorsa daha genel bir abstraction seç.

İlk hedef mükemmel sistem değil.

İlk hedef:

**Çalışan, temiz ve genişletilebilir ORH çekirdeği.**
