# ORH Projesi — Oturum Raporu

Bu doküman, bu chat oturumunda ORH (Open Runtime for AI Orchestration) projesi üzerinde planlanan ve gerçekleştirilen tüm işlerin özetidir.

---

## 1. Manifesto Çalışması

- `manifesto.md` (kısa/İngilizce taslak) ve `techincal_manifesto.md` (detaylı/Türkçe) okundu ve karşılaştırıldı.
- **techincal_manifesto.md'ye yeni bir prensip eklendi: "8. Sistem Taşınabilirliği"**
  - Özet: Önceden kurulmuş bir agentic sistem (custom framework, LangChain, CrewAI, AutoGen...) kara kutu olarak sarılmak zorunda kalmadan ORH'ye taşınabilmeli. Sistemin orkestrasyon mantığı ORH'nin `Component`/`Event`/`Command` primitive'lerine gerçek şekilde yeniden ifade edilebilmeli; sadece dile/kütüphaneye özgü yürütme detayları (örn. Selenium) bridge üzerinden dışarıda bırakılabilir.
  - Bu prensip, `Component`/`Event`/`Command` tasarımının neden bu kadar genel tutulduğunu haklı çıkaran bir kısıt olarak konuldu.

---

## 2. Kullanıcının BrowserAgent Projesi (Bağlam)

- `/home/rifat/Masaüstü/BrowserAgent` incelendi: LangChain **değil**, kullanıcının kendi yazdığı bir agent framework'ü ("Mimar" orkestratör + `sosyal_medya_agent`/`content_creator_agent`/`browser_agent` alt-ajanları, OpenAI-compatible tool-calling, Selenium/Playwright tabanlı browser tool'ları).
- Tartışılan soru: "Bu sistem ORH'ye taşınabilir mi?" → Cevap: kara kutu olarak değil, ama tool-calling karar döngüsü ORH'nin native component'i olarak yeniden yazılabilir; Python tool'ları (Selenium vs.) bir bridge/MCP server üzerinden çağrılabilir. Bu tartışma daha sonra **Faz 5**'in (toolbox + MCP) gerçek gerekçesi oldu.

---

## 3. Faz 1 — Temel Runtime (daha önceki oturumda tamamlanmıştı, bu oturumda doğrulandı)

- `.orh` formatı, parser, validator, `AgentComponent`, Ollama provider, CLI (`init`/`validate`/`run`).
- Bu oturumda: Go kurulumu doğrulandı (`/usr/local/go`), gerçek Ollama ile uçtan uca test tekrarlandı (`orh init` → `orh validate` → `orh run` → doğru cevap).

---

## 4. Faz 2 — Event-Driven Graph Runtime

- **Yeni primitive'ler:** `Event{Source,Target,Port,Payload any}`, `Command` (açık interface: `EmitEvent`, `SetOutput`), `Component` (tek metodlu, değişmedi).
- **Scheduler:** gerçek event-queue tabanlı (FIFO), fan-out'u doğal destekliyor, cycle'lara `MaxEvents` güvenlik sınırıyla dayanıklı (validator cycle'ı reddetmiyor, runtime sonsuz döngüden korunuyor).
- **CLI:** `orh inspect` (zincir diyagramı / düz liste), `orh run --debug` (Event/Executing/Output/Completed trace).
- **Paket reorganizasyonu:** `internal/runtime/{events,scheduler,executor}`, `internal/graph`, `internal/components`, `internal/commands`.
- **Doğrulama:** sequential 2'li zincir, fan-out, eski tek-ajan senaryosu — hepsi gerçek Ollama ile test edildi.

---

## 5. Faz 3 — GitHub Paket Sistemi

- `orh.yaml` manifest formatı (`spec.Manifest`, `Dependencies` hem `[]` hem `{}` formunu kabul ediyor).
- `internal/pkg` — manifest + entrypoint doğrulama.
- `internal/cache` — `~/.orh/cache/github/{owner}/{repo}/{commit}` (`ORH_HOME` ile test edilebilir).
- `internal/lock` — `orh.lock`.
- `internal/resolver/github` — ref parse, GitHub API üzerinden default branch/commit çözümü, tarball indirme + **zip-slip korumalı** açma.
- `internal/resolver` — `Pull` (indir/cache/validate/dependency'leri özyinelemeli çöz, circular dependency koruması), `Installed`, `GetInfo`.
- **CLI:** `orh pull`, `orh list`, `orh info`, `orh validate`/`orh run` paket dizini ve uzak referans (`owner/repo[@version]`) desteği.
- **Gerçek GitHub testi:** `Ahmet2001/orh-test-package` reposu yayınlandı, `orh pull` + `orh run` ile gerçek Ollama'ya karşı uçtan uca çalıştırıldı. **Bu repo kullanıcının isteğiyle canlı bırakıldı.**

---

## 6. Faz 4 — Architecture Composition

- **Mimari düzeltme (kullanıcı onayıyla):** Faz4.md'nin önerdiği ayrı `Executable` interface'i (Inputs()/Outputs()/Execute()) reddedildi. Onun yerine kompozisyon tamamen **build-time graph rewrite** olarak tasarlandı — runtime'da yeni bir component tipi yok.
- `internal/composition` — `Flatten()`: `type: orchestration` component'lerini özyinelemeli olarak açıp (`prefix.` ile component/model adlarını yeniden adlandırıp) tek düz bir graph'a dönüştürüyor. Named-port kontratı (`inputs:`/`outputs:`) ve port'suz eski paketlerle geriye dönük uyumluluk destekleniyor. Circular dependency tespiti var.
- **Bug fix:** `graph.SplitEndpoint` "ilk nokta"dan "son nokta"ya göre bölecek şekilde değiştirildi (flatten sonrası component adları da nokta içerdiği için, örn. `search.crawler.input`).
- Validator'a yön kontrolleri (`output` kaynak olamaz, `input` hedef olamaz) port'lu endpoint'leri de kapsayacak şekilde güncellendi.
- **Gerçek GitHub testi:** `Ahmet2001/orh-test-subagent` (named-port kontratlı) yayınlandı, yerel bir `compose-test` mimarisi onu `type: orchestration` ile içe aktardı, flatten edilmiş graph gerçek Ollama ile doğru çalıştı ("Tokyo" cevabı).

---

## 7. Faz 5 — Toolbox / MCP / Tool-Calling

- **Karar noktaları (kullanıcı onayıyla):**
  - `Init()`/`Shutdown()` yerine önce sadece tool-calling için `Component` interface'i **değiştirilmedi** — `AgentComponent` genişletildi (`Tools` boşsa eski davranış, doluysa `ChatProvider` ile tool-calling döngüsü).
  - Permission sistemi **bilinçli olarak gevşek** tutuldu — manifest'te `permissions` alanı var ama sadece deklaratif, hiçbir zorlayıcı kontrol yok (kullanıcının açık isteğiyle: "güvenlik kurallarını katı yapmaya gerek yok").
- `internal/mcp` — minimal MCP client (stdio üzerinden JSON-RPC 2.0: `initialize`, `tools/list`, `tools/call`), gerçek subprocess ve testte in-memory pipe ile çalışıyor.
- `internal/toolbox` — `.tb` dosya formatı parse/validate.
- `internal/tools` — `Tool` interface, `MCPTool`, `Registry`.
- `internal/pkg` — `kind: toolbox` desteği eklendi (paketler artık ya mimari ya toolbox olabiliyor).
- `providers.ToolCallingProvider` — yeni `Chat()` arayüzü; Ollama'nın native `/api/chat` uç noktası (tool şeması + `tool_calls` destekli) üzerinden implemente edildi.
- `executor.BuildToolRegistry` — paketin `orh.yaml` dependencies'inden toolbox'ları çekip MCP server'ları başlatıyor, tool'ları `alias.toolName` olarak kaydediyor (sadece `tools:` kullanan mimarilerde devreye giriyor).
- **CLI:** `orh tools list`, `orh tool test`, `orh info` toolbox desteği.
- **Bug fix (test sırasında bulundu):** MCP server'ın çalışma dizini ayarlanmamıştı (`runtime.server` göreli path'li komutlar yanlış dizinden çalışıyordu) — `mcp.Start`'a `dir` parametresi eklendi.
- **Gerçek uçtan uca doğrulama:** Python'da yazılmış minimal bir MCP server (`add` tool'u) içeren `Ahmet2001/orh-test-toolbox` yayınlandı; bunu kullanan `Ahmet2001/orh-test-tool-user` mimarisi gerçek Ollama (`qwen3:1.7b`) ile büyük sayılar toplayarak (847293+592817=1.440.110, doğru) tool'un gerçekten çağrıldığı kanıtlandı. `orh tool test` ile LLM'siz doğrudan tool çağrısı da doğrulandı.

---

## 8. Faz 6 — Custom Node Sistemi (WASM)

- **Karar noktaları (kullanıcı onayıyla, uygulamaya geçmeden önce netleştirildi):**
  1. `Component` interface'i bozulmadı — `Init()`/`Shutdown()` **opsiyonel** interface'ler (`Initializer`/`Shutdowner`) olarak eklendi, `io.Closer` deseni gibi. `AgentComponent` hiç dokunulmadı.
  2. WASM runtime kütüphanesi: **wazero** (saf Go, CGo yok) seçildi.
  3. Node SDK dili: **Go/TinyGo** seçildi (Rust yerine) — proje tek dilde kalsın diye. Sadece node *yazan* geliştiriciler TinyGo'ya ihtiyaç duyuyor, kullanan kimse hiçbir ek araç kurmuyor.
  4. Host API v1 kapsamı başta **daraltılmıştı**: `orh.emit` (dönüş değeri üzerinden implicit), `orh.log`, `orh.state.get/set` var; `orh.call_model`/`orh.call_tool` ilk iterasyonda ertelenmişti. Kullanıcı sonradan **"enforcement ile birlikte ekleyemez misin"** diyerek bunu tekrar açtı — aşağıda "Faz 6.1" olarak ayrıca anlatılıyor.
- **Araç kurulumu:** TinyGo 0.41.1 + uyumlu Go 1.26.7 (`/usr/local`, TinyGo Go 1.27'yi henüz desteklemiyor).
- `sdk/node` — guest-tarafı SDK, **bilinçli olarak `internal/` dışına** kondu (Go'nun internal paket kısıtı, dış node yazarlarının onu import etmesini engellerdi). Kendi `go.mod`'u var (ayrı, düşük Go sürümü — ana projenin `go 1.27.0` gereksinimine bağımlı olmasın diye).
- `internal/wasm` — wazero host wrapper (`Module.Load`/`Handle`/`Close`), host fonksiyonları (`orh_log`, `orh_state_get`, `orh_state_set`).
- **Çözülen gerçek teknik sorun:** TinyGo'nun `wasip1` hedefi WASI "command" modeli üretiyor; `_start` çalışıp `proc_exit` çağırınca wazero modülü kalıcı olarak kapatıyordu (`handle` ikinci kez çağrılamıyordu). Çözüm: WASI'nin `proc_exit` fonksiyonu no-op ile override edildi.
- `internal/nodes` — `.node` manifest parse/validate, `NodeComponent` (Component + Initializer + Shutdowner), `Load()`.
- `internal/pkg` — `kind: node` desteği (`.wasm` dosyasının varlığı + WASM magic byte kontrolü ile hafif doğrulama).
- Scheduler'a `Init`/`Shutdown` lifecycle çağrıları ve **panic-recovery** eklendi (bir node çökerse ORH process'i çökmüyor, hata olarak dönüyor — ama tam "retry/fallback/stop/ignore" mimari karar mekanizması bu fazın kapsamı dışında bırakıldı).
- **CLI:** `orh nodes list`, `orh node test`, `orh node build` (TinyGo'yu shell'den çağırıyor), `orh info` node desteği.
- **Gerçek uçtan uca doğrulama:** TinyGo ile derlenmiş gerçek bir `smart-router` node'u içeren `Ahmet2001/orh-test-node` yayınlandı. Onu `type: node` ile kullanan bir mimari, payload'a göre `agent_a`/`agent_b`'ye dinamik yönlendirme yaptı — `"urgent"` girdisi sandbox içinden `orh.log` ile loglanıp doğru şekilde `agent_b`'ye yönlendirildi, gerçek Ollama'dan doğru cevap alındı.

### 8.1 Faz 6.1 — `orh_call_model` / `orh_call_tool` (permission enforcement ile)

Kullanıcının "enforcement ile birlikte ekleyemez misin" talebi üzerine, host API'ye iki fonksiyon daha eklendi — bu sefer `.node` manifest'indeki `permissions.models.allow`/`permissions.tools.allow` **gerçekten** uygulanıyor (Faz5'teki deklaratif-kalsın kararının aksine, kullanıcı bunun için enforcement istedi).

- `internal/wasm/host.go`: `orh_call_model`/`orh_call_tool` host fonksiyonları eklendi. Tek dönüş değeriyle başarı/hata ayrımı yapılamadığı için `callResult{ok,result,error}` JSON zarfı kullanıldı (guest bunu çözüp hata mı başarı mı olduğuna bakıyor).
- `internal/wasm/module.go`: `HostAPI` interface'i `CallModel(ctx, modelSlot, prompt)` / `CallTool(ctx, toolName, argsJSON)` ile genişletildi.
- `internal/nodes/component.go`: `NodeComponent`'e `Models map[string]spec.Model`, `Tools *tools.Registry`, `Permissions spec.NodePermissions` alanları eklendi. `CallModel`/`CallTool` metodları **önce izin listesine bakıyor** (`contains(...)`), sonra model/tool'a gidiyor — reddedilen çağrı ağa hiç çıkmıyor.
- `internal/providers/resolve` (yeni paket): `providers` ile `providers/ollama` arasındaki import-cycle'ı önlemek için `resolve.Provider`/`resolve.ChatProvider` buraya taşındı; `executor.go` da aynı fonksiyonları kullanacak şekilde refactor edildi (kod tekrarı kalmadı).
- `internal/runtime/executor/executor.go`: `buildNodeComponent` artık mimarinin `models:` bloğunu ve `tools.Registry`'i node'a enjekte ediyor. `needsTools` genişletildi: bir node bileşeninin tool kullanıp kullanmayacağı statik olarak bilinemediği için (o bilgi `.node` manifest'inde, paket çekilene kadar görünmez), herhangi bir `type: node` bileşeni artık tool registry'nin kurulmasını tetikliyor.
- `sdk/node/node.go`: guest tarafına `CallModel(modelSlot, prompt string) (string, error)` ve `CallTool(name, argsJSON string) (string, error)` eklendi.
- **Testler:** `internal/nodes/component_test.go` (izin verilen/verilmeyen model-slot ve tool senaryoları, fake registry ile), `internal/wasm/callable_test.go` (gerçek derlenmiş bir WASM fixture'ı — `testdata/caller` — üzerinden ABI'nin ptr/len + JSON zarfının uçtan uca çalıştığını kanıtlıyor, hem başarı hem ret senaryosu).
- **Gerçek uçtan uca doğrulama (4 senaryo, `Ahmet2001/orh-test-node` v2.0.0):** node'un `.node` manifest'i `permissions.models.allow: ["reasoning"]` ve `permissions.tools.allow: ["math.add"]` deklare ediyor.
  1. İzinli model slotu (`reasoning`) → gerçek Ollama'ya (`qwen3:1.7b`) gidip "Fransa'nın başkenti" sorusuna **"Paris"** cevabı geldi.
  2. İzinsiz model slotu (`other`) → ağa hiç çıkmadan `"model slot \"other\" is not permitted"` hatasıyla reddedildi.
  3. İzinli tool (`math.add`, `Ahmet2001/orh-test-toolbox`'daki gerçek MCP sunucusu) → `{"a":17,"b":25}` girdisiyle gerçek MCP server'dan **42** cevabı geldi.
  4. İzinsiz tool (`math.subtract`) → tool registry'e hiç gitmeden `"tool \"math.subtract\" is not permitted"` hatasıyla reddedildi.

---

## 8.2 Faz 6.2 — `kind: skill` (yeniden kullanılabilir prompt/yetenek paketleri)

Kullanıcı "system prompt ve skill için bir şey ekledik mi" diye sordu; cevap hayırdı, ardından Claude Code'daki skill kavramının ORH karşılığını istedi. Önce açık soruldu: "topluluktan biri kendi custom node yapısıyla bunu zaten oluşturabilir miydi?" — cevap evet (node'lar `Component` arayüzünü paylaştığı için payload'ı sarıp bir agent'a önden bağlamak zaten mümkündü), ama bu üç pratik sorunu var: her prompt değişikliği için WASM yeniden derleme gerekir, çoklu skill'ler manuel node-wiring gerektirir, ve keşfedilebilirlik formalize değil. Bu yüzden native, birinci sınıf bir `kind: skill` paket türü eklendi.

- **Yeni paket türü:** `kind: skill`, entrypoint bir `SKILL.md` dosyası — opsiyonel YAML frontmatter (`description:`) + gövde (talimat metni). `internal/skill` paketi bunu parse ediyor (`---` sınırlayıcılarla).
- `internal/pkg/pkg.go`: `KindSkill` eklendi, `Package.Skill *spec.Skill` alanı, `Validate()`'e yeni bir `case KindSkill` dalı.
- `internal/spec/spec.go` + `internal/spec/skill.go`: `Component.Skills []SkillRef` — kullanıcı "path referansı mı yoksa direkt mi yazılıyor, ikisi de olmalı" dediği için **iki mod da destekleniyor**, tek bir `skills:` listesinde karışık kullanılabiliyor:
  - Düz string → dependency alias (`- review`), GitHub'daki bir `kind: skill` paketine referans
  - Mapping → `text:` alanıyla **doğrudan `.orh` dosyasının içine yazılan** talimat (`- text: |` + opsiyonel `description:`), hiçbir paket/bağımlılık gerektirmiyor
  `SkillRef.UnmarshalYAML` bu iki şekli ayırt ediyor (scalar node → Alias, mapping node → Inline).
- `internal/runtime/executor/skills.go` (yeni): `needsSkills` (bir component `skills:` deklare ediyorsa true) ve `BuildSkillPrompts(ctx, deps)` — her skill bağımlılığını GitHub'dan çekip (`toolbox`'daki `BuildToolRegistry` ile aynı desen, ama MCP sunucu başlatma yok — skill'in kendi runtime'ı yok, sadece metin) alias -> talimat metni haritası döndürüyor.
- `executor.go`: `buildComponents`, agent'ın `Prompt`'unu inşa ederken her `skills:` girdisinin talimat metnini `\n\n` ile ekliyor (skill talimatı bulunamazsa "references undefined skill" hatası — `tools:`'daki aynı desenle tutarlı).
- **Testler:** `internal/skill/skill_test.go` (frontmatter parse, kapanmamış frontmatter hatası, boş instructions validasyonu), `internal/pkg/pkg_test.go` (skill paketi yükleme/doğrulama), `internal/parser/parser_test.go` (YAML'daki iki `skills:` şeklinin de — alias string ve `text:` mapping — doğru parse edildiği, `text:` alanı eksikse hata verdiği), `internal/runtime/executor/executor_test.go` (`buildComponents` ile: alias-only, inline-only, ve **karışık** (`[alias, inline]` aynı listede) senaryoları — hepsi ağa çıkmadan, senkronize sahte veriyle).
- **Gerçek uçtan uca doğrulama (iki mod da ayrı ayrı test edildi):**
  1. **Paket-referans modu:** `Ahmet2001/orh-test-skill` (v1.0.0) yayınlandı — `SKILL.md`: "her zaman tek kelime, tamamen BÜYÜK HARF cevap ver". Aynı agent + aynı soru ("Almanya'nın başkenti neresi?"), gerçek Ollama (`qwen3:1.7b`) karşısında: skill **bağlıyken** `"BERLIN"`, skill **bağlı değilken** (kontrol grubu) `"The capital of Germany is **Berlin**..."` (tam cümle).
  2. **Inline mod:** hiçbir `orh.yaml`/paket olmadan, sadece bir `.orh` dosyası içine doğrudan yazılan `skills: [{text: "...tek kelime BÜYÜK HARF..."}]` ile — "İtalya'nın başkenti neresi?" sorusuna gerçek Ollama'dan **"ROME"** cevabı geldi (hiç ağ/paket çekme gerekmedi).

---

## 9. Genel Mimari Prensip (tüm fazlar boyunca korunan)

Her fazda tekrar tekrar uygulanan kural: **ORH herhangi bir orchestration/tool/node türünü özel olarak (hardcoded) implement etmez.** Tüm yeni yetenekler (composition, tool-calling, WASM node'lar) mevcut `Component`/`Event`/`Command` primitive'lerinin üzerine, mümkün olduğunca **yeni interface eklemeden** (sadece Go'nun opsiyonel-interface deseniyle) inşa edildi. Faz4'te önerilen `Executable` interface'i ve Faz6'da önerilen zorunlu 3-metodlu `Component` interface'i bu yüzden bilinçli olarak reddedilip daha sade alternatiflerle değiştirildi.

---

## 10. Yayınlanan Test Repoları (GitHub'da canlı)

| Repo | Amaç | Faz |
|---|---|---|
| `Ahmet2001/orh-test-package` | Temel paket testi (kullanıcı isteğiyle kalıcı) | 3 |
| `Ahmet2001/orh-test-subagent` | Named-port kontratlı composition testi | 4 |
| `Ahmet2001/orh-test-toolbox` | MCP tool-calling testi (`add` tool'u) | 5 |
| `Ahmet2001/orh-test-tool-user` | Toolbox'ı kullanan mimari | 5 |
| `Ahmet2001/orh-test-node` | WASM smart-router node'u (v2.0.0: `call_model`/`call_tool` + enforcement testi) | 6 / 6.1 |
| `Ahmet2001/orh-test-skill` | `kind: skill` testi ("tek kelime BÜYÜK HARF" talimatı) | 6.2 |

---

## 11. Kurulan Araçlar (sistem seviyesinde)

- Go 1.27.0 → `/usr/local/go`
- Go 1.26.7 (TinyGo uyumluluğu için) → `/usr/local/go1.26`
- TinyGo 0.41.1 → `/usr/local/tinygo`
- `orh` CLI binary → `~/.local/bin/orh`

---

## 12. Açık / Ertelenmiş İşler

- ~~`orh.call_model()` / `orh.call_tool()` host fonksiyonları~~ — **Faz 6.1'de tamamlandı** (bkz. §8.1), enforcement dahil.
- **Faz6'nın tam hata-yönetimi mimarisi** (Retry/Fallback/Stop/Ignore, ErrorEvent routing) — sadece "node crash ORH'yi çökertmez" kısmı yapıldı, mimarinin kendi kararı verebileceği bir routing mekanizması yok.
- Manifest'teki `permissions` alanları — Faz6'daki `models.allow`/`tools.allow` artık gerçekten enforce ediliyor (§8.1); Faz5'in `network`/`filesystem` alanları ve Faz6'nın `network`/`filesystem` alanları hâlâ sadece deklaratif (kullanıcının bilinçli tercihi, henüz genişletilmedi).
- Faz3'te konuşulan "paket dışarıda çalışan bir servise ihtiyaç duyuyor" bildirimi (manifest'e `requires.services` gibi bir alan) — Faz5/6 ile kısmen (`dependencies` + toolbox/node kind'leri) karşılandı ama resmi bir `requires` alanı hâlâ yok.

---

## 13. Tüm Kod Değişiklikleri

Tüm değişiklikler `/home/rifat/Masaüstü/OrhestrationCLI/orh/` altında, gerçek `go build`/`go vet`/`go test ./...` ile her fazda doğrulandı — hepsi yeşil. Her faz sonunda birim testleri + en az bir gerçek uçtan uca senaryo (gerçek Ollama, çoğu fazda gerçek GitHub, Faz5'te gerçek MCP server, Faz6'da gerçek WASM sandbox) ile doğrulandı, uydurma/varsayımsal sonuç raporlanmadı.
