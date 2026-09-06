# ORH Manifesto

## Open Runtime for AI Orchestration

Yapay zeka sistemleri artık tek bir modelden ibaret değildir.

Geleceğin AI sistemleri; farklı modellerin, agent'ların, araçların, hafıza sistemlerinin ve özel bileşenlerin birlikte çalıştığı dinamik yapılardan oluşacaktır.

Bugün modelleri çalıştırmak kolaylaşmıştır. Kullanıcılar birkaç komutla yerel modeller indirebilir, çalıştırabilir ve kendi sistemlerine entegre edebilir.

Ancak aynı kolaylık yapay zeka mimarileri için henüz oluşmamıştır.

Bir agent sistemi kurmak için geliştiricilerin çoğu zaman sıfırdan kod yazması, karmaşık framework'ler öğrenmesi ve kendi altyapısını oluşturması gerekir.

ORH bu problemi çözmek için vardır.

---

# Vizyon

ORH'nin amacı:

**Yapay zeka mimarilerini, modeller kadar kolay paylaşılabilir, indirilebilir ve çalıştırılabilir hale getirmektir.**

Nasıl ki bir kullanıcı:

```bash
ollama pull model
ollama run model
```

ile bir modeli kullanabiliyorsa;

ORH ile:

```bash
orh pull architecture
orh run architecture
```

ile bir yapay zeka mimarisini kullanabilmelidir.

---

# Temel Felsefe

## 1. Architecture is Code

Yapay zeka mimarileri gizli, kapalı ve tekrar yazılması gereken yapılar olmamalıdır.

Bir kişinin oluşturduğu:

* araştırma agent sistemi,
* kodlama sistemi,
* tartışma sistemi,
* karar verme sistemi,
* otomasyon sistemi

başkaları tarafından yeniden kullanılabilir olmalıdır.

Bir AI architecture, paylaşılabilir bir yazılım bileşeni olmalıdır.

---

# 2. ORH Architecture Dayatmaz

ORH belirli orchestration modellerine bağlı değildir.

Sequential, Debate, Supervisor, Swarm veya gelecekte ortaya çıkacak yeni mimariler ORH'nin içinde sabit özellikler değildir.

Bunlar ORH runtime üzerinde çalışan yapılardır.

ORH sadece temel yapı taşlarını sağlar:

* Component
* Event
* State
* Connection
* Execution
* Dependency

Bu sayede geliştiriciler kendi hayal ettikleri mimarileri oluşturabilir.

---

# 3. Model Bağımsızlığı

Bir architecture belirli bir modele ait değildir.

Bir mimari:

```text
Research Agent
        |
        |
   Reasoning Model
```

tanımlar.

Hangi modelin kullanılacağı kullanıcıya aittir.

Aynı architecture:

* Ollama,
* vLLM,
* OpenAI uyumlu API'ler,
* farklı yerel modeller

ile çalışabilmelidir.

ORH'nin görevi modelleri seçmek değil, modelleri birlikte çalıştırmaktır.

---

# 4. Açık Ekosistem

ORH kapalı bir platform değildir.

Dünyanın herhangi bir yerindeki geliştirici:

* kendi architecture'ını,
* kendi toolbox'ını,
* kendi component'ini

oluşturabilir ve paylaşabilir.

Bir kullanıcı:

```bash
orh pull developer/project
```

ile dünyanın başka bir yerindeki AI sistemini kullanabilir.

---

# 5. Composition First

Geleceğin AI sistemleri tek bir büyük uygulama olmayacaktır.

Küçük, yeniden kullanılabilir parçaların birleşiminden oluşacaktır.

Bir developer:

* başkasının research architecture'ını,
* başka birinin search toolbox'ını,
* başka birinin reasoning component'ini

birleştirerek yeni bir sistem oluşturabilmelidir.

ORH'nin temel prensibi:

**Her şey birleşebilir olmalıdır.**

---

# 6. GitHub Native Community

ORH kendi ekosistemini merkezi bir depolama sistemine bağımlı kurmaz.

Architecture'lar açık kaynak depolama sistemlerinde yaşayabilir.

Başlangıçta:

* GitHub repository'leri,
* Git versioning,
* release/tag sistemi

source of truth olarak kullanılabilir.

ORH yalnızca keşif, doğrulama ve çalıştırma katmanı sağlar.

---

# 7. Güvenli Çalıştırma

Açık ekosistem güvenlik gerektirir.

Bir AI component'i:

* hangi araçlara erişebilir,
* hangi modellere bağlanabilir,
* hangi kaynakları kullanabilir

açık şekilde tanımlamalıdır.

ORH, capability-based izin modeli ile güvenli ve kontrollü execution sağlamayı hedefler.

---

# 8. Sistem Taşınabilirliği

ORH'den önce kurulmuş bir agentic sistem, kara kutu olarak sarılmak zorunda kalmadan ORH'ye taşınabilmelidir.

Bir geliştirici:

* kendi yazdığı özel (custom) bir agent framework'ünü,
* LangChain ile kurduğu bir orchestration'ı,
* CrewAI, AutoGen veya benzeri bir sistemi

ORH'ye getirdiğinde, o sistemin karar mantığı (hangi tool'un ne zaman çağrılacağı, hangi adımın sıradaki adımı belirleyeceği) ORH'nin kendi component/graph modeli içinde gerçek şekilde ifade edilebilmelidir.

Yani hedef:

**Sistemi tek bir opak servis çağrısına indirgemek değil, sistemin orkestrasyon mantığını ORH'nin primitive'leri (Component, Event, Command) üzerinden yeniden ifade edebilmektir.**

Bunun karşılığında, o sistemin dil veya kütüphaneye özgü kısımları (örneğin Python'a bağımlı bir browser automation aracı) yeniden yazılmak zorunda değildir; ince bir execution bridge üzerinden ORH'nin çağırabileceği bir yeteneğe dönüştürülebilir.

Bu ayrım nettir:

* **Orkestrasyon mantığı** → ORH'nin native primitive'lerine taşınır.
* **Framework'e/dile özgü yürütme detayları** → bridge üzerinden dışarıda bırakılabilir.

Bu prensip ORH'nin temel tasarım kısıtlarından biridir:

`Component`, `Event` ve `Command` soyutlamaları, herhangi bir dışarıdan gelen agentic sistemin orkestrasyon mantığını bu primitive'lere yeniden yazılabilecek kadar genel kalmalıdır. Bir primitive, yalnızca ORH'nin kendi component'lerine yetecek kadar dar tasarlanamaz — dışarıdan bir sistemi (LangChain, CrewAI, custom framework) taşımaya çalışan birinin önünde bir engel oluşturuyorsa, o tasarım hâlâ eksiktir.

---

# ORH Runtime Prensipleri

ORH runtime:

* architecture'ları yükler,
* dependency'leri çözer,
* execution graph oluşturur,
* event'leri yönetir,
* component yaşam döngülerini kontrol eder.

Ancak architecture'ın kendisini sınırlamaz.

Runtime küçük ve genel kalır.

Ekosistem ise topluluk tarafından büyür.

---

# Uzun Vadeli Hedef

ORH sadece bir orchestration framework değildir.

Uzun vadede hedef:

**Yapay zeka sistemleri için açık, birleşebilir ve topluluk tarafından geliştirilen bir çalışma ortamı oluşturmaktır.**

Bugün:

```text
Model
```

indiriyoruz.

Yarın:

```text
Architecture
```

indireceğiz.

Sonrasında:

```text
AI System
```

oluşturacağız.

---

# ORH

Open Runtime for AI Orchestration

Models are the intelligence.

Architectures are the systems.

ORH is the layer that connects them.
