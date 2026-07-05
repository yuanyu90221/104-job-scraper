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

`.github/workflows/` 底下有 4 個教學 workflow，示範 Pants 的依賴圖查詢、全量 build、增量 build，以及
「開 PR 自動判斷學習關卡並留言」的互動式導師（`04-pr-mentor.yml`）。另外 `pants-ci.yml` 是實際跑在
每次 push / PR 上的 CI 守門。

完整教學（每個 Step 在教什麼、如何解讀 Mermaid 依賴圖、建議操作順序、Codespaces 練習環境）都整理在
**[專案 Wiki](https://github.com/yuanyu90221/104-job-scraper/wiki)**，README 只留這個入口，避免內容越滾越長：

- **[Pants Build 教學指南](https://github.com/yuanyu90221/104-job-scraper/wiki/Pants-Build-教學指南)** — Step 1~3 手動教學 workflow 詳解
- **[PR 導師 Workflow](https://github.com/yuanyu90221/104-job-scraper/wiki/PR-導師-Workflow)** — Step 4 自動化互動教學、關卡判斷規則
- **[GitHub Codespaces 練習環境](https://github.com/yuanyu90221/104-job-scraper/wiki/GitHub-Codespaces-練習環境)** — 免安裝的互動式練習環境

## 每日自動爬蟲

`daily-scrape.yml` 每天 01:00 UTC（台灣時間 09:00）自動執行，也可以手動 `workflow_dispatch` 並自訂 `keyword` / `months` / `pages` / `line_top`。流程：checkout → build → 執行爬蟲 → 透過 LINE Messaging API 把前 `line_top` 筆結果推播出去（預設 10）→ 把完整結果 `jobs.json` 存成 30 天效期的 Artifact。推播需要三個 GitHub Secrets：`LINE_CHANNEL_SECRET`、`LINE_CHANNEL_ACCESS_TOKEN`、`LINE_TARGET_ID`。
