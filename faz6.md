Sen kıdemli bir sistem mimarı ve Go geliştiricisisin.

Üzerinde çalıştığımız proje:

# ORH (Open Runtime for AI Orchestration)

ORH, yapay zeka architecture'larını tanımlamak, paylaşmak, çalıştırmak ve birbirine bağlamak için açık kaynak bir runtime'dır.

---

# Mevcut Durum

## Faz 1

Tamamlandı:

* `.orh` formatı
* Agent component
* Ollama provider
* CLI

## Faz 2

Tamamlandı:

* Event-driven runtime
* Graph execution
* Scheduler
* Component bağlantıları

## Faz 3

Tamamlandı:

* GitHub package sistemi
* Architecture download
* Cache
* Lock sistemi

## Faz 4

Tamamlandı:

* Architecture composition
* Sub-orchestration
* Architecture as component

## Faz 5

Tamamlandı:

* Toolbox (`.tb`)
* Tool registry
* MCP entegrasyonu
* Agent tool calling
* Permission sistemi

---

# Faz 6 Ana Hedef

ORH artık sadece hazır component'leri kullanan bir sistem olmamalıdır.

Geliştiriciler kendi execution component'lerini oluşturabilmelidir.

Yeni hedef:

> Community tarafından geliştirilen güvenli ve taşınabilir AI component ekosistemi oluşturmak.

---

# Yeni Package Type: Node

Yeni package türü:

```yaml
kind: node
```

Örnek repository:

```text
github.com/rifat/smart-router


orh.yaml

router.node

README.md

```

---

# Node Kavramı

Node:

ORH runtime içinde çalışan özel bir component'tir.

Örnekler:

* Router
* Planner
* Memory Manager
* Validator
* Evaluator
* Data Transformer
* Custom Agent Logic

---

# Component Model

Tüm ORH parçaları aynı temel interface'e sahip olmalıdır.

```go
type Component interface {


    Init(
        ctx Context,
    ) error


    Handle(
        ctx Context,
        event Event,
    ) ([]Command,error)


    Shutdown(
        ctx Context,
    ) error

}

```

Agent:

```text
Component
```

Tool:

```text
Component
```

Node:

```text
Component
```

Architecture:

```text
Component
```

olmalıdır.

---

# WASM Runtime

Custom node'lar doğrudan host üzerinde çalıştırılmamalıdır.

Yanlış:

```text
ORH

 |

random binary

```

Doğru:

```text
ORH Runtime

      |

 WASM Sandbox

      |

 Custom Node

```

---

# WASM Host

ORH WASM component'lerine kontrollü API sağlamalı.

Örnek:

```text
orh.emit()

orh.state.get()

orh.state.set()

orh.call_model()

orh.call_tool()

orh.log()

```

Node:

```text
WASM

```

sadece izin verilen host function'ları çağırabilir.

---

# Node SDK

Community geliştiricileri için SDK oluştur.

İlk SDK:

Go veya Rust.

Örnek API:

```rust
use orh_sdk::*;


struct Router;


impl Node for Router {


    fn execute(
        input: Value
    ) -> Value {


        // custom logic

    }

}

```

Build:

```bash
orh node build

```

Çıktı:

```text
router.wasm

```

---

# Node Manifest

Örnek:

```yaml
apiVersion: orh/v1


kind: node


name: smart-router


version: 1.0.0


runtime:

  type: wasm


entrypoint:

  module: router.wasm


permissions:


  models:

    - reasoning


  tools:

    - web.search


  network:false


```

---

# Node Input / Output Contract

Her node açık interface belirtmeli.

Örnek:

```yaml
inputs:


 task:

   type:string



outputs:


 destination:

   type:string

```

Runtime:

```text
Input

 |

Node

 |

Output

```

---

# Architecture İçinde Node Kullanımı

Örnek:

main.orh:

```yaml
components:


 router:


   type: node


   source:

      github:rifat/smart-router



 researcher:


   type: agent



connections:


 input

   -> router.input


router.output

   -> researcher.input

```

---

# Node Dependency

Architecture:

```yaml
dependencies:


 router:

   source:

      github:rifat/smart-router

```

diyebilir.

Resolver:

```text

Architecture

      |

Dependencies

      |

Node Packages

      |

WASM Modules

```

yüklemeli.

---

# Permission Sistemi Genişlet

Node'lar daha güçlü olduğu için izin sistemi genişletilmeli.

Manifest:

```yaml
permissions:


 filesystem:


   read:

      - /data


 network:


   allow:


      - api.example.com


 models:


   allow:


      - reasoning


 tools:


   allow:


      - web.search


```

---

# Runtime Isolation

Bir node hata verdiğinde:

Yanlış:

```text
Node crash

↓

ORH crash

```

Doğru:

```text
Node crash

↓

Component failure event

↓

Runtime continues

```

---

# Error Event Sistemi

Yeni event:

```go
type ErrorEvent struct {


 component string


 error string


}

```

Architecture karar verebilmeli:

```text
Retry

Fallback

Stop

Ignore

```

---

# CLI Komutları

## Node listele

```bash
orh nodes list

```

Örnek:

```text
Installed nodes:


smart-router

memory-manager

verifier

```

---

## Node info

```bash
orh info rifat/smart-router

```

---

## Node test

```bash
orh node test smart-router

```

---

## Node build

```bash
orh node build

```

---

# Test Senaryosu

Custom router node oluştur.

Repository:

```text
github.com/test/router-node

```

İçerik:

```text
orh.yaml

router.node

router.wasm

```

Architecture:

```text
Input

 |

Router Node

 |

Agent A

or

Agent B

```

Çalıştır:

```bash
orh run router-test

```

Beklenen:

```text
Loading node:

smart-router


Initializing WASM runtime


Node ready


Event received


Routing decision:

agent-b


Execution continues

```

---

# Kod Yapısı

Yeni klasörler:

```
internal/


nodes/

    loader.go

    registry.go


wasm/

    runtime.go

    host.go


sdk/


    node/


permissions/


    policy.go

```

---

# Faz 6 Başarı Kriteri

Faz sonunda:

Bir geliştirici:

1. Kendi node'unu yazabilir.

2. WASM olarak build edebilir.

3. GitHub'a koyabilir.

4. Başka kullanıcı:

```bash
orh pull developer/custom-node

```

ile indirebilir.

5. Architecture içinde:

```yaml
type: node

```

olarak kullanabilir.

---

# Tasarım Prensibi

ORH'nin temel prensibi artık:

```
Everything is a Component.

Components are composable.

Components are shareable.

Components are sandboxed.

```

Modeller intelligence sağlar.

Architecture reasoning sağlar.

Tool'lar capability sağlar.

Node'lar extensibility sağlar.

ORH bunların çalışma katmanıdır.
