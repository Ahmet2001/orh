Sen kıdemli bir sistem mimarı ve Go geliştiricisisin.

Üzerinde çalıştığımız proje:

# ORH (Open Runtime for AI Orchestration)

ORH, yapay zeka architecture'larını tanımlamak, paylaşmak, çalıştırmak ve birbirine bağlamak için açık kaynak bir runtime'dır.

---

# Mevcut Durum

## Faz 1 tamamlandı:

* `.orh` formatı mevcut.
* Agent component mevcut.
* Ollama provider mevcut.
* Basit runtime çalışıyor.

## Faz 2 tamamlandı:

* Event-driven graph runtime mevcut.
* Birden fazla component bağlanabiliyor.
* Graph execution çalışıyor.

## Faz 3 tamamlandı:

* GitHub package resolver mevcut.
* `orh pull user/repo` çalışıyor.
* Architecture package sistemi mevcut.
* Manifest (`orh.yaml`) mevcut.
* Local cache ve lock sistemi mevcut.

---

# Faz 4 Ana Hedef

ORH artık architecture'ların birbirini kullanabildiği bir **composable AI architecture runtime** haline gelmelidir.

Bir `.orh` dosyası başka bir `.orh` package'ını import edebilmelidir.

Örneğin:

Bir kullanıcı:

```text
research-system.orh
```

içinde:

```text
research-agent
+
debate-agent
+
fact-check-agent

```

birleştirebilmelidir.

---

# Temel Prensip

Bir architecture aynı zamanda bir component olabilir.

Yani:

```
Component
    |
    |
    +---- Agent
    |
    +---- Tool
    |
    +---- Architecture

```

ORH runtime açısından:

```
Simple Agent Architecture

          |
          |
          v

Large Research Architecture

```

aynı interface'i kullanmalıdır.

---

# Sub-Orchestration Desteği

Yeni component tipi ekle:

```yaml
type: orchestration
```

Örnek:

```yaml
components:


 research:

   type: orchestration

   source:
     github:ayse/research-agent

   version:
     v1.0.0


 judge:

   type: agent

   model:
     main

```

---

# Input / Output Contract

Her architecture açık bir interface tanımlamalıdır.

Örnek:

```yaml
inputs:


 question:

   type: string



outputs:


 answer:

   type: string

```

Başka architecture bunu kullanırken:

```yaml
connections:


 - from: input.question

   to: research.question



 - from: research.answer

   to: judge.input

```

yapabilmeli.

---

# Architecture Interface

Yeni abstraction oluştur:

```go
type Executable interface {


    Inputs() []Port


    Outputs() []Port


    Execute(
        ctx context.Context,
        input Event,
    ) ([]Event,error)


}

```

Hem:

```text
AgentComponent

```

hem:

```text
OrchestrationComponent

```

bu interface'i uygulayabilmeli.

---

# Nested Runtime

Bir architecture çalıştırılırken:

Ana runtime:

```
Main Runtime


    |
    |
    +---- Research Runtime

              |
              |
              +---- Agent A
              +---- Agent B


    |
    |
    +---- Debate Runtime

              |
              |
              +---- Agent C

```

oluşturabilmeli.

Ama ayrı process olmak zorunda değil.

İlk aşamada:

aynı runtime içinde nested execution yeterli.

---

# Dependency Resolution

Bir architecture başka architecture import ettiğinde:

Örnek:

```yaml
dependencies:


 research:

   source:
     github:ayse/research


 debate:

   source:
     github:rifat/debate


```

Runtime:

```
Main Architecture

        |
        |
        +---- Resolve dependencies

                    |
                    |
              Load packages

                    |
                    |
              Build graph

                    |
                    |
              Execute

```

yapmalı.

---

# Graph Flattening

İki yaklaşım olabilir.

İlk implementasyon:

Architecture import edildiğinde graph içine açılabilir.

Örnek:

Önce:

```
Research Component

```

Sonra:

```
Research Component

    |
    |
    +---- Search Agent

    |
    |
    +---- Summarizer Agent

```

haline gelir.

Runtime bunu tek graph olarak çalıştırır.

---

# Alternative: Black Box Execution

Architecture içeriği gizli de olabilir.

Örnek:

```
Main Graph


      Research Module


```

içeride:

```
Research Runtime


A
B
C

```

Bu ileride:

* sandbox
* remote execution
* cloud execution

için kullanılabilir.

İlk fazda flatten yaklaşımı yeterli.

---

# Yeni CLI Komutları

## Dependency gösterme

```bash
orh deps username/project
```

Örnek:

```
rifat/deep-research


├── ayse/search-agent
├── mert/debate
└── ali/fact-check

```

---

## Architecture inspect

```bash
orh inspect username/project
```

Çıktı:

```
Architecture:

deep-research


Components:

✓ research-agent
✓ debate
✓ judge


Dependencies:

✓ ayse/search-agent
✓ mert/debate

```

---

# Circular Dependency Kontrolü

Eklenmeli:

Örnek:

A:

```
depends B

```

B:

```
depends A

```

hata vermeli:

```
Circular dependency detected

```

---

# Version Uyumluluğu

Basit semver desteği ekle:

Örnek:

```yaml
dependencies:

 debate:

   version:

      "^1.0"

```

Destek:

```
major
minor
patch

```

---

# Cache Davranışı

Nested dependency'ler de cache edilmeli.

Örnek:

```
~/.orh/cache/


github/

    rifat/

        deep-research/

             commit1


    ayse/

        search-agent/

             commit4

```

---

# Test Senaryosu

3 architecture oluştur:

## Package 1

```
search-agent

input:
 query

output:
 documents

```

## Package 2

```
summarizer

input:
 documents

output:
 summary

```

## Package 3

```
research-system


search-agent

+

summarizer

```

Çalıştır:

```bash
orh run research-system

```

Beklenen:

```
Loading research-system

Resolving dependencies

Loading search-agent

Loading summarizer


Building graph


Executing


Completed

```

---

# Kod Yapısı

Yeni eklemeler:

```
internal/


composition/

    resolver.go

    loader.go


components/


    orchestration.go


runtime/


    nested.go


ports/

    contract.go


dependency/

    graph.go

```

---

# Faz 4 Başarı Kriteri

Faz sonunda:

Bir kullanıcı:

Başka birinin architecture'ını:

```bash
orh pull user/research-agent

```

kendi architecture'ına:

```yaml
components:

 research:

   type: orchestration

   source:
      user/research-agent

```

olarak ekleyebilmeli.

Ve:

```bash
orh run my-system

```

dediğinde:

```
My System

    |
    |
Research Architecture

    |
    |
Debate Architecture

    |
    |
Final Agent

```

tek runtime tarafından çalıştırılmalı.

---

# Sonraki Faz

Faz 5:

* `.tb` toolbox sistemi
* MCP entegrasyonu
* tool discovery
* agent tool kullanımı
* permission modeli

olacaktır.

Ana prensip:

"An ORH architecture is not just a workflow.

It is a reusable AI system component."
