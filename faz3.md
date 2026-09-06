Sen kıdemli bir sistem mimarı ve Go geliştiricisisin.

Üzerinde çalıştığımız proje:

# ORH (Open Runtime for AI Orchestration)

ORH, yapay zeka orchestration mimarilerini tanımlamak, paylaşmak ve çalıştırmak için açık kaynak bir runtime'dır.

---

# Mevcut Durum

Faz 1 tamamlandı:

* `.orh` dosyası parse ediliyor.
* Agent component çalışıyor.
* Ollama provider mevcut.
* CLI mevcut.

Faz 2 tamamlandı:

* Event-driven runtime mevcut.
* Graph execution mevcut.
* Birden fazla component bağlanabiliyor.
* Component → Event → Scheduler → Component akışı çalışıyor.
* Sequential ve basit graph yapıları `.orh` ile oluşturulabiliyor.

---

# Faz 3 Ana Hedef

ORH artık lokal bir runtime olmaktan çıkıp **paylaşılabilir architecture package sistemi** haline gelmelidir.

Kullanıcı:

```bash
orh pull user/project
```

dediğinde GitHub üzerindeki ORH architecture paketini indirebilmeli.

Daha sonra:

```bash
orh run user/project
```

ile çalıştırabilmeli.

---

# Temel Tasarım Kararı

ORH kendi merkezi storage sistemini oluşturmayacak.

Source of truth:

GitHub repository'leri olacak.

Yani:

```text
GitHub Repository

        |
        |
        v

ORH Resolver

        |
        |
        v

Local Cache

        |
        |
        v

ORH Runtime

```

---

# Package Concept

Her ORH architecture bir package'dır.

Örnek repository:

```
github.com/rifat/debate-pro

├── orh.yaml
├── main.orh
├── prompts/
│   ├── critic.md
│   └── judge.md
└── README.md

```

---

# Manifest Sistemi

Her package kökünde:

```
orh.yaml
```

olmalı.

Örnek:

```yaml
apiVersion: orh/v1

kind: orchestration

name: debate-pro

version: 1.0.0


entrypoint: main.orh


author:

  github: rifat


dependencies: []

```

---

# Package Validator

Yeni validation ekle.

Kontroller:

* orh.yaml var mı?
* apiVersion destekleniyor mu?
* kind geçerli mi?
* entrypoint mevcut mu?
* entrypoint valid `.orh` dosyası mı?
* dependency formatı doğru mu?

Komut:

```bash
orh validate .
```

Çıktı:

```
✓ manifest valid
✓ entrypoint found
✓ architecture valid
```

---

# GitHub Resolver

Yeni bir resolver oluştur.

Destek:

```bash
orh pull rifat/debate-pro
```

Bunu:

```
github.com/rifat/debate-pro
```

olarak çöz.

Ayrıca destekle:

```bash
orh pull github:rifat/debate-pro
```

---

# Version Desteği

Destek:

Latest:

```bash
orh pull rifat/debate-pro
```

Tag:

```bash
orh pull rifat/debate-pro@v1.2.0
```

Commit:

```bash
orh pull rifat/debate-pro@a83bc91
```

---

# Lock Sistemi

Dependency reproducibility için:

```
orh.lock
```

oluştur.

Örnek:

```yaml
lockVersion: 1


packages:

  github:rifat/debate-pro:

    ref: v1.2.0

    commit: a83bc91

```

Amaç:

Aynı architecture her zaman aynı dependency ile çalışmalı.

---

# Local Cache

Cache sistemi oluştur.

Örnek:

```
~/.orh/


cache/

    github/

        rifat/

            debate-pro/

                a83bc91/


config/

runs/

```

---

# CLI Komutları

Yeni komutlar ekle:

## Pull

```bash
orh pull user/project
```

Görev:

* GitHub'dan indir
* validate et
* cache'e koy

---

## List

```bash
orh list
```

Örnek:

```
Installed architectures:

rifat/debate-pro
ayse/research-agent

```

---

## Info

```bash
orh info rifat/debate-pro
```

Örnek:

```
Name:
debate-pro


Version:
1.2.0


Author:
rifat


Components:
5


Dependencies:
0

```

---

## Run Remote Architecture

Destek:

```bash
orh run rifat/debate-pro
```

Akış:

```
resolve package

↓

load cache

↓

read manifest

↓

load entrypoint

↓

compile graph

↓

execute runtime

```

---

# Dependency Sistemi

Architecture başka architecture kullanabilmeli.

Örnek:

```yaml
dependencies:


 debate:

   source:
     github:rifat/debate-pro

   version:
     v1.2.0

```

İlk fazda sadece dependency resolve et.

Nested execution zorunlu değil.

---

# Package Model

İç yapıyı gelecekte:

```
orh.yaml

main.orh

components/

tools/

nodes/

```

destekleyebilecek şekilde tasarla.

Ama Faz 3 sadece:

```
orh.yaml

+
main.orh

```

destekleyecek.

---

# CLI UX Hedefi

Kullanıcı deneyimi:

Model tarafı:

```bash
ollama pull llama3
ollama run llama3

```

Architecture tarafı:

```bash
orh pull rifat/debate-pro
orh run rifat/debate-pro

```

Aynı hissiyat oluşturulmalı.

---

# Kod Yapısı

Yeni eklemeler:

```
internal/

resolver/

    github/

package/

    manifest.go

    loader.go

cache/

    manager.go

lock/

    lockfile.go

cli/

    pull.go

    info.go

    list.go

```

---

# Güvenlik

Bu fazda:

* sadece `.orh` çalıştır.
* arbitrary code çalıştırma.
* shell execution yok.

Çünkü custom node sistemi daha sonra gelecek.

---

# Test Senaryosu

GitHub'da örnek repo:

```
github.com/test/simple-agent

```

İçerik:

```
orh.yaml

main.orh

```

Komut:

```bash
orh pull test/simple-agent

```

Beklenen:

```
Downloading package...

✓ manifest valid

✓ architecture cached


```

Sonra:

```bash
orh run test/simple-agent

```

Beklenen:

```
Loading architecture

Creating graph

Executing runtime

Completed

```

---

# Faz 3 Başarı Kriteri

Faz sonunda:

Bir kullanıcı:

1. GitHub'da ORH architecture repository oluşturabilir.

2. Başka kullanıcı:

```bash
orh pull username/repository
```

ile indirebilir.

3. Şunu çalıştırabilir:

```bash
orh run username/repository
```

ve architecture lokal runtime'da çalışır.

---

# Sonraki Faz

Faz 4:

* sub-orchestration
* architecture composition
* dependency execution
* toolbox hazırlığı
* MCP entegrasyon altyapısı

olacaktır.

Geliştirme boyunca şu prensibi koru:

"ORH package'ları model değil, AI architecture dağıtır."

"GitHub storage'dır, ORH runtime'dır."

"ORH runtime architecture'ı çalıştırır, architecture runtime'ı tanımlamaz."
