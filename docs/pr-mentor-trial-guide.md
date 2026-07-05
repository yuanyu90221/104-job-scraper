# 如何手動體驗 04-pr-mentor.yml（不改程式碼也能觸發）

`04-pr-mentor.yml` 是 `pull_request`（`opened` / `synchronize`）事件觸發的教學 workflow，
本篇記錄一套最小步驟：從 `main` 開一個測試分支、開 PR 觸發導師留言、看完就把 PR 關掉，
全程不需要合併，也不一定要真的改動程式碼。

## 步驟一：從 main 開一個新分支

```bash
git checkout main
git pull origin main
git checkout -b demo/pr-mentor-trial
```

## 步驟二：產生一個 commit 讓分支跟 main 不同

GitHub 不允許對完全相同的 commit 開 PR，所以分支上至少要有一個不同的 commit。
如果不想真的改任何檔案，用空 commit 就夠了：

```bash
git commit --allow-empty -m "test: trigger pr-mentor demo (no file changes)"
git push -u origin demo/pr-mentor-trial
```

> 這個 PR 就算完全沒改檔案，也一定會落在**關卡一（全量 build）**——因為
> `04-pr-mentor.yml` 是先看這個 PR 編號專屬的快取（`pants-lmdb-tutorial-pr<PR
> 編號>-...`）有沒有命中，新 PR 一定是空快取，跟有沒有真的改
> `internal/notifier`／`internal/models` 無關。想體驗關卡二、三，需要在同一個
> 分支上再 push 真的改這兩個套件的 commit（見 README「如何使用」段落）。

## 步驟三：開 PR，觸發導師留言

```bash
gh pr create --base main --head demo/pr-mentor-trial \
  --title "test: pr-mentor workflow trial (no code change)" \
  --body "純測試 04-pr-mentor.yml，不改任何檔案，跑完就關閉不合併"
```

PR 一開起來，`04-pr-mentor.yml` 會自動被觸發；等它跑完，PR 底下會出現一則
sticky comment（同一個 PR 只維護一則留言，之後 push 新 commit 會更新內容，不
會愈刷愈多則）。

## 步驟四：看完留言，關閉 PR（不合併）

```bash
gh pr close <PR編號>
```

不想留著測試分支的話，可以連分支一起刪：

```bash
gh pr close <PR編號> --delete-branch
```

`--delete-branch` 會同時刪除遠端與本地追蹤分支，是不可逆操作，確認不需要這個
分支再下這個指令。
