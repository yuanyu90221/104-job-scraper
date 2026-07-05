# 104 Job Scraper

爬取 104 人力銀行的職缺搜尋 API，篩選近期刊登的職缺，並可將結果輸出成表格 / JSON / CSV，或透過 LINE Messaging API 彙整前 N 筆推播出去。

本專案同時也是一個**學習 Pants Build 如何整合進 GitHub Actions** 的教材：repo 裡的 `.github/workflows/` 用一組「教學 workflow」示範全量 build、增量 build、依賴圖查詢等 Pants 核心概念，可以直接在 GitHub Actions 頁面手動觸發觀察結果。

## 專案目的

- 提供一個好用的 CLI，依關鍵字 / 地區 / 天數搜尋 104 職缺（預設關鍵字 `golang 後端工程師`）。
- 每天定時（`daily-scrape.yml`）自動爬取並透過 LINE Messaging API 推播最新職缺。
- 用真實的 Go module 當範例，示範 [Pants](https://www.pantsbuild.org/) 這套建置工具如何做到「只重新編譯有變更的套件」，並把這個概念用 GitHub Actions + Mermaid 圖視覺化出來。

## 專案架構

```
cmd/main.go          CLI 入口（cobra）
internal/search      呼叫 104 API + 分頁邏輯
internal/client      HTTP client
internal/models      Job / SearchResponse 資料結構
internal/formatter   輸出格式化（table / json / csv）
internal/notifier    LINE Messaging API 推播
scripts/             輔助腳本（patch-pants-go.sh）
```

依賴方向：`cmd` → `search` → `client` → `models`；`cmd` → `formatter`、`notifier`（`notifier` 也依賴 `models`）。

## 安裝與使用

```bash
go build -o bin/104-job-scraper ./cmd/main.go

./bin/104-job-scraper \
  --keyword="golang 後端工程師" \
  --months=3 \
  --pages=15 \
  --format=table
```

常用參數：

| 參數 | 說明 |
| --- | --- |
| `--keyword, -k` | 搜尋關鍵字（預設 `golang 後端工程師`） |
| `--area, -a` | 地區代碼，留空代表全台灣 |
| `--days, -d` | API 端幾天內刊登（0/3/7/14/30） |
| `--months, -m` | 客戶端過濾：只保留幾個月內刊登的職缺 |
| `--pages, -p` | 最多爬取頁數（每頁 20 筆） |
| `--format, -f` | 輸出格式：`table` / `json` / `csv` |
| `--output, -o` | 輸出到檔案，預設印到 stdout |
| `--line-channel-secret` | LINE Messaging API channel secret（搭配 `--line-channel-token` 設定後自動推播；預設讀取環境變數 `LINE_CHANNEL_SECRET`） |
| `--line-channel-token` | LINE Messaging API channel access token（預設讀取環境變數 `LINE_CHANNEL_ACCESS_TOKEN`） |
| `--line-target-id` | 推播目標的 LINE `userId` 或 `groupId`（預設讀取環境變數 `LINE_TARGET_ID`） |
| `--line-top` | 推播到 LINE 的職缺筆數（預設 10） |

三個 LINE 參數都可以不用手動輸入，直接把 credentials 放進 `.env` 後匯出成環境變數即可（明確傳入的 flag 仍會覆蓋環境變數）：

```bash
set -a && source .env && set +a
./bin/104-job-scraper --months=3
```

## 用這個專案學習 Pants Build 的 GitHub Action

`.github/workflows/` 底下有 3 個「教學 Step」workflow，都是 `workflow_dispatch`（手動觸發），可以在 GitHub 網頁的 **Actions** 分頁逐一執行，每次執行都會在 Job Summary 產生 Mermaid 圖解釋當前發生的事：

1. **`01-pants-list.yml` — 展示 Targets 與依賴圖**
   執行 `pants list ::`、`pants dependencies` 查看專案裡所有 Pants target，以及 `search`、`cmd` 的依賴關係圖。適合第一次認識「Pants 如何看待一個 Go module」。

2. **`02-full-build.yml` — 全量 Build（無快取）**
   不還原 LMDB 快取，直接 `pants test ::` + `pants package cmd:bin`，示範「完全沒有快取」時每個套件都要重新編譯，並記錄耗時。

3. **`03-incremental.yml` — 增量 Build（有快取）**
   還原 LMDB 快取後，用 `pants --changed-since=HEAD~1 --changed-dependents=transitive` 只測試「有變更 + 受影響」的套件。手動觸發時可用 `changed_package` 這個 input 選擇模擬變更 `formatter` / `client` / `models` / `notifier`，觀察依賴圖顏色（綠=快取命中、紅=重新編譯、橘=因依賴而重建）如何不同（改 `models` 會牽動全部套件，改葉節點只影響自己 + `cmd`）。

4. **`04-pr-mentor.yml` — PR 導師（自動、互動式）**
   前三個 Step 都要手動觸發；這一個是 `pull_request` 事件觸發（開 PR / push 新 commit 時自動跑），
   用**規則式腳本**（不呼叫 LLM）判斷你目前處在哪個學習關卡，並在 PR 底下留言（同一個 PR 只維護
   一則留言，後續 push 會更新內容，不會愈刷愈多則）：
   - PR 第一次執行、Pants 快取還是空的 → **關卡一**：全量 build。
   - 改了 `internal/notifier`（或 `internal/formatter`）→ **關卡二**：只有它跟 `cmd` 需要重建。
   - 改了 `internal/models` → **關卡三**：因為 fan-in 最高，全部套件都被牽動重建。
   - 改了其他檔案（如只改 `cmd`／文件）→ 通用留言，引導你去試上面兩個檔案。

   判斷依據是真實的 `pants --changed-since=<base_sha> --changed-dependents=transitive list` 輸出跟
   真實的 `pants test` 耗時，不是模擬數據；快取 key 也是每個 PR 各自獨立（`pants-lmdb-tutorial-pr<PR
   編號>-...`），所以每個新 PR 都會重新體驗一次「關卡一」的冷啟動。

   **如何使用（不用手動觸發，開 PR 就會自動跑）：**
   1. 從 `main` 開一個新分支，push 上去後對著 `main` 開一個 PR（不需要任何額外設定）。
   2. PR 一開起來，`04-pr-mentor.yml` 會自動被 `pull_request` 事件觸發並開始執行；等它跑完，
      PR 底下會自動出現一則導師留言（第一次通常是「關卡一：全量 build」，因為這個 PR 的快取還是空的）。
   3. 在同一個分支上繼續修改程式碼並 push 新 commit：
      - 只改 `internal/notifier/line.go`（或 `internal/formatter`）→ 下次留言會更新成「關卡二」。
      - 改 `internal/models/job.go` → 下次留言會更新成「關卡三」（fan-in 最高，全部套件重建）。
   4. 不用手動刷新或重新觸發任何東西——留言是同一則「sticky comment」，每次 push 完 workflow 跑完就會自動更新內容，不會愈刷愈多則。
   5. 想重新從關卡一開始體驗，開一個全新的 PR 即可（快取 key 是每個 PR 各自獨立的）。

   > 想要不改任何程式碼、只是快速跑一次看看導師留言長什麼樣子？參考
   > [`docs/pr-mentor-trial-guide.md`](docs/pr-mentor-trial-guide.md)，裡面示範用空
   > commit 開一個測試用 PR，看完留言就關閉、不用合併。

另外 `pants-ci.yml` 是實際跑在每次 push / PR 上的 CI：一個 job 用純 `go test`/`go vet`/`go build` 把關，另一個 job 用 Pants 重跑一次（`pants list ::` + `pants package cmd:bin`），驗證 Pants 設定本身沒有壞掉。

### 建議的操作順序

1. Fork 或 clone 這個 repo 到你自己的 GitHub 帳號。
2. 到 **Actions** 分頁，依序手動執行 `教學 Step 1` → `Step 2` → `Step 3`。
3. 每個 workflow 跑完後點進 run 頁面看 **Summary**，裡面的 Mermaid 圖會搭配當次的實際 log 說明發生了什麼事。
4. 對照 Step 2（全量、無快取）跟 Step 3（增量、有快取）的耗時輸出，感受 Pants 快取帶來的差異。
5. 想自己做實驗，有兩種方式：
   - 手動：修改 `internal/formatter` 或 `internal/models` 裡的檔案後 push 一個 commit，再手動跑一次
     Step 3，觀察 `pants --changed-since` 抓到的變更範圍是否符合預期。
   - 自動／互動：改用 Step 4 的方式——直接開一個 PR，依序修改 `internal/notifier/line.go`、
     `internal/models/job.go`，每次 push 完不用手動做任何事，看 PR 底下的導師留言自動更新，一路從
     關卡一走到關卡三。

### 用 GitHub Codespaces 建立互動式練習環境

repo 已內建 `.devcontainer/devcontainer.json`，開一個 Codespace 就會自動裝好 Go 1.25 + Pants，並套用 Go 1.24+ 的 `GOEXPERIMENT` workaround（`scripts/patch-pants-go.sh`）—— 跟教學 workflow 裡的 `Install Pants` / `Patch Pants Go backend` 兩步驟完全一致，不用手動再貼指令：

1. 到 repo 頁面點 **Code → Codespaces → Create codespace on main**。
2. 等 Codespace 建立完成（`postCreateCommand` 會自動跑完安裝 + patch），開終端機直接執行：

```bash
pants list ::
pants dependencies internal/search::
pants --changed-since=HEAD~1 --changed-dependents=transitive list
```

這幾個指令跟 `01-pants-list.yml` / `03-incremental.yml` 在 CI 裡跑的完全相同，差別只是在 Codespace 裡是互動式的，可以隨時改 `internal/formatter` 或 `internal/models` 的檔案後立刻重跑 `--changed-since` 看差異，不用等 GitHub Actions run 完。

## 每日自動爬蟲

`daily-scrape.yml` 每天 01:00 UTC（台灣時間 09:00）自動執行，也可以手動 `workflow_dispatch` 並自訂 `keyword` / `months` / `pages` / `line_top`。流程：checkout → build → 執行爬蟲 → 透過 LINE Messaging API 把前 `line_top` 筆結果推播出去（預設 10）→ 把完整結果 `jobs.json` 存成 30 天效期的 Artifact。推播需要三個 GitHub Secrets：`LINE_CHANNEL_SECRET`、`LINE_CHANNEL_ACCESS_TOKEN`、`LINE_TARGET_ID`。
