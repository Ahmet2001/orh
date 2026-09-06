Sen kıdemli bir sistem mimarı ve Go geliştiricisisin.

Üzerinde çalıştığımız proje:

# ORH (Open Runtime for AI Orchestration)

ORH, yapay zeka architecture'larını tanımlamak, paylaşmak, çalıştırmak ve birbirine bağlamak için açık kaynak bir runtime'dır.

---

# Mevcut Durum

## Faz 1

Tamamlandı:

* `.orh` parser
* Agent component
* Ollama provider
* CLI

## Faz 2

Tamamlandı:

* Event-driven graph runtime
* Multiple component execution
* Connections
* Scheduler

## Faz 3

Tamamlandı:

* GitHub package resolver
* Architecture packages
* `orh pull`
* Cache
* Lock sistemi

## Faz 4

Tamamlandı:

* Architecture composition
* Sub-orchestration
* Architecture as component
* Dependency graph

---

# Faz 5 Ana Hedef

ORH architecture'ları artık dış araçları kullanabilmelidir.

Bunun için:

* Toolbox package sistemi (`.tb`)
* Tool interface
* Tool discovery
* Agent tool calling
* Permission sistemi

geliştirilecektir.

---

# Temel Tasarım Prensibi

Tool'lar architecture'ın içine gömülü olmamalıdır.

Yanlış:

```yaml
agent:

  tools:

    - custom_search_function

```

Çünkü bu paylaşılabilirliği bozar.

Doğru:

Tool'lar ayrı package olarak tanımlanmalıdır.

Örnek:

```text
github.com/rifat/web-tools


orh.yaml

web.tb

tools/

```

Başka architecture bunu import eder.

---

# Yeni Package Türü: Toolbox

Yeni kind ekle:

```yaml
kind: toolbox
```

Örnek:

```yaml
apiVersion: orh/v1

kind: toolbox


name: web-tools

version: 1.0.0


entrypoint: web.tb


author:

  github: rifat

```

---

# .tb Formatı

Yeni dosya:

```
web.tb
```

Örnek:

```yaml
version: 1


tools:


  search:


    description:
      Search the web


    input:


      query:
        type: string


    output:


      results:
        type: array


    runtime:

      type: mcp

      server:
        web-search


```

---

# Tool Interface

ORH içinde generic tool interface oluştur.

Örnek:

```go
type Tool interface {


    Name() string


    Description() string


    InputSchema() Schema


    Execute(
        ctx context.Context,
        input Value,
    ) (Value,error)


}

```

---

# Agent Tool Kullanımı

Agent component artık tool kullanabilmeli.

Örnek architecture:

```yaml
components:


 researcher:


   type: agent


   model: main


   tools:


     - web.search


```

Runtime:

```text
User Question

      |

Agent

      |

Need more information

      |

Call Tool

      |

Tool Result

      |

Continue reasoning

      |

Final Output

```

---

# Tool Calling Event Model

Tool çağrısı event tabanlı olmalı.

Örnek:

Agent:

```text
ToolCallEvent

{
 tool:
   web.search

 arguments:
   {
    query:"latest AI news"
   }
}

```

Runtime:

```text
ToolCallEvent

        |

Tool Executor

        |

ToolResultEvent

        |

Agent

```

---

# MCP Entegrasyonu

İlk tool provider olarak MCP destekle.

Model Context Protocol zaten tool keşfi ve çağrısı için uygun bir standarttır.

ORH:

```text
ORH Runtime

      |

MCP Client

      |

MCP Server

      |

Tools

```

---

# MCP Tool Discovery

Bir toolbox:

```yaml
runtime:

  type:mcp

  server:
    github-search

```

başlatıldığında:

ORH:

1. MCP server'a bağlanır.
2. Available tools listesini alır.
3. Tool registry'e ekler.

Örnek:

```text
Available tools:

✓ github.search
✓ github.read_file
✓ github.issue_create

```

---

# Tool Registry

Runtime içinde:

```go
type ToolRegistry struct {


 tools map[string]Tool


}

```

Örnek:

```text
web.search

github.read

database.query

```

---

# Architecture İçinde Toolbox Kullanımı

Örnek:

orh.yaml:

```yaml
dependencies:


 web:

   source:
     github:rifat/web-tools


```

main.orh:

```yaml
components:


 researcher:


   type: agent


   tools:


     - web.search

```

---

# Permission Sistemi

Tool'lar güvenli çalışmalıdır.

Manifest:

```yaml
permissions:


 network:


   allow:

     - google.com


 filesystem:


   read:false


```

---

# Runtime Permission Check

Bir agent:

```
web.search
```

çağırır.

Runtime:

```text
Checking permissions...

Tool:
web.search


Network:
allowed


Execute:
yes

```

---

# Tool Cache

Tool sonuçları cache edilebilir.

Örneğin:

```text
cache/

 tools/

   web.search/

      hash(input)

          result

```

---

# CLI Yeni Komutları

## Toolbox listeleme

```bash
orh tools list
```

Örnek:

```text
Installed tools:


web.search

github.read

database.query

```

---

## Toolbox info

```bash
orh info rifat/web-tools

```

Çıktı:

```text
Type:
toolbox


Tools:

✓ search
✓ fetch


Runtime:

MCP

```

---

## Tool test

```bash
orh tool test web.search

```

---

# Kod Yapısı

Yeni klasörler:

```
internal/


tools/

    registry.go

    executor.go


toolbox/

    parser.go

    loader.go


mcp/

    client.go


permissions/

    checker.go

```

---

# Test Senaryosu

Bir toolbox oluştur:

```
github.com/test/web-tools

```

İçinde:

```
web.tb

```

Tool:

```
search

```

Architecture:

```
research-agent


Agent:

uses web.search

```

Çalıştır:

```bash
orh run research-agent

```

Beklenen:

```text
Loading architecture

Loading toolbox

Connecting MCP server

Available tools:

✓ web.search


Agent started


Tool call:

web.search


Tool result received


Final answer generated

```

---

# Faz 5 Başarı Kriteri

Faz sonunda:

Bir kullanıcı:

1. GitHub'da toolbox oluşturabilir.

2. Başka biri:

```bash
orh pull user/toolbox
```

ile indirebilir.

3. Bir architecture:

```yaml
tools:

 - user.toolbox.tool

```

ile kullanabilir.

4. Agent runtime sırasında tool çağırabilir.

---

# Sonraki Faz

Faz 6:

* Custom Node sistemi
* WASM runtime
* Plugin SDK
* Community tarafından yazılmış component'ler

olacaktır.

Ana prensip:

"Models provide intelligence.

Architectures provide reasoning structure.

Tools provide capabilities.

ORH connects them."
