# File Browser 自訂版維運手冊（Merge / 編譯 / 部署）

本文件是給目前這個 fork 使用，目標是讓你之後可以固定用同一套流程：
- 把創作者（`upstream`）更新合併進你的 repo（`origin`）
- 重新編譯前端 + 後端
- 部署到你的 Docker 環境
- 出問題時可快速回滾

---

## 0. 前置假設

- 你的 repo 有兩個 remote：
  - `origin` = 你自己的 fork
  - `upstream` = `filebrowser/filebrowser`
- 你現在使用的部署方式是：
  - Docker image 仍用 `filebrowser/filebrowser:latest`
  - 但用 volume 把自編譯 binary 掛進容器的 `/bin/filebrowser`

---

## 1. 每次同步上游（upstream）更新

以下流程以 `master` 為主分支。

```bash
cd /path/to/your/filebrowser
git checkout master

# 先拿到最新遠端資訊
git fetch origin
git fetch upstream --tags

# 保證本地 master 先對齊你自己的 origin/master
git pull --ff-only origin master

# 把 upstream/master 合併進來
git merge upstream/master
```

如果有衝突：

```bash
# 看衝突檔案
git status

# 手動修完後
git add <conflicted-files>
git commit
```

合併完成後推回自己的 fork：

```bash
git push origin master
```

---

## 2. 編譯（前端 + 後端）

### 2.1 編譯前端

```bash
cd /path/to/your/filebrowser/frontend
pnpm install --frozen-lockfile
pnpm run build
```

### 2.2 編譯後端（把最新 frontend embed 進 binary）

回到 repo 根目錄：

```bash
cd /path/to/your/filebrowser
go build -o filebrowser-custom .
chmod +x filebrowser-custom
```

如果部署目標是 Linux（常見 NAS/路由器容器）建議明確指定：

```bash
cd /path/to/your/filebrowser
GOOS=linux GOARCH=amd64 go build -a -o filebrowser-custom .
chmod +x filebrowser-custom
```

`GOARCH` 依目標機器調整（例如 `amd64` / `arm64`）。

---

## 3. 部署到你目前的 Docker Compose 架構

## 3.1 放置 binary

把剛編好的檔案放到你實際掛載的位置，例如：

```bash
cp /path/to/your/filebrowser/filebrowser-custom /mnt/vio4-1/Configs/filebrowser/filebrowser-custom
chmod +x /mnt/vio4-1/Configs/filebrowser/filebrowser-custom
```

## 3.2 Compose 關鍵設定（重點）

你必須把 host 的 custom binary 掛到容器的 `/bin/filebrowser`，例如：

```yaml
services:
  filebrowser:
    image: filebrowser/filebrowser:latest
    volumes:
      - /mnt/vio4-1/Configs/filebrowser/filebrowser-custom:/bin/filebrowser:ro
```

如果掛到其他路徑（例如 `/filebrowser`）通常不會生效，因為容器實際執行的是 `/bin/filebrowser`。

## 3.3 套用更新

```bash
docker compose up -d --force-recreate filebrowser
```

---

## 4. 部署後驗證（建議每次都做）

```bash
# 1) 檢查容器內 binary 是否可執行
docker compose exec filebrowser /bin/filebrowser version

# 2) 比對 host/container 的 binary hash 是否一致
sha256sum /mnt/vio4-1/Configs/filebrowser/filebrowser-custom
docker compose exec filebrowser sha256sum /bin/filebrowser

# 3) 看前端 manifest（確認是不是新 hash）
curl -s http://<你的IP>:8082/static/manifest.json
```

另外瀏覽器建議做一次 hard refresh，避免舊快取影響判斷。

---

## 5. 建議的日常更新節奏

每次上游有更新時，固定跑以下順序：

1. `git fetch upstream --tags`
2. `git checkout master && git pull --ff-only origin master`
3. `git merge upstream/master`
4. `pnpm run build`（在 `frontend/`）
5. `go build -o filebrowser-custom .`
6. 複製 binary 到 `/mnt/vio4-1/Configs/filebrowser/`
7. `docker compose up -d --force-recreate filebrowser`
8. 跑第 4 節驗證

---

## 6. 回滾方案（強烈建議保留上一版 binary）

部署前先備份上一版：

```bash
cp /mnt/vio4-1/Configs/filebrowser/filebrowser-custom /mnt/vio4-1/Configs/filebrowser/filebrowser-custom.bak
```

新版本若異常，立即回滾：

```bash
cp /mnt/vio4-1/Configs/filebrowser/filebrowser-custom.bak /mnt/vio4-1/Configs/filebrowser/filebrowser-custom
chmod +x /mnt/vio4-1/Configs/filebrowser/filebrowser-custom
docker compose up -d --force-recreate filebrowser
```

---

## 7. 常見問題

### Q1: 已經 `pnpm run build`，但畫面還是舊版？
- 原因通常是後端 binary 沒重編，或容器沒跑到新 binary。
- `filebrowser` 的前端是 embed 在 Go binary 裡，不是只看 `frontend/dist`。

### Q2: `strings filebrowser-custom | grep "某字串"` 找不到？
- 這個檢查不可靠。前端資產會 minify + gzip，很多字串不會直接出現在 `strings`。
- 請用第 4 節的 hash 與 manifest 驗證。

### Q3: 目前 compose 還是 `image: filebrowser/filebrowser:latest`，這樣可以嗎？
- 可以，只要你正確把 custom binary 掛到 `/bin/filebrowser`。
- 若要更可控，未來可改成自建 image（binary 打進 image），部署會更一致。
