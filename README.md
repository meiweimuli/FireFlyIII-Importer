# 🔥 FireFly III Importer

> 🤖 **Note:** This project, including its code and documentation, was entirely generated and built by AI (Gemini / Antigravity).

A robust and powerful tool to effortlessly import **Alipay (CSV)** and **WeChat Pay (XLSX)** transaction records into your [Firefly III](https://www.firefly-iii.org/) instance.

## ✨ Features

- **Drag & Drop Upload:** Seamlessly drag your Alipay `.csv` and WeChat Pay `.xlsx` bills directly into the UI.
- **Advanced Mapping Engine:**
  - Build complex hierarchical rules with **AND/OR** logic and **NOT** negations.
  - Automatically map categories, asset accounts, expense/revenue accounts, and modify transaction types based on custom conditions.
  - Test and preview rules locally before importing.
- **Real-Time Import Streaming:** View the live progress of your import. Pause or abort mid-import if needed.
- **Smart Deduplication:** Verifies external IDs directly with the Firefly III server to prevent duplicate transactions—even across different import batches.
- **Safe Previews:** The UI presents a fully parsed, chronological list of transactions prior to submission. You can bulk-edit, reset, or selectively import specific rows.
- **Configuration Management:** Export and import your mapping rules as JSON, with automatic server-side backups for peace of mind.
- **Docker Ready:** Deploy anywhere easily.

## 🚀 Quick Start (Docker)

The easiest way to run FireFly III Importer is via Docker.

```yaml
# docker-compose.yml
version: "3.8"
services:
  firefly-importer:
    image: kuons/fireflyiii-importer:latest # Or build from source
    # build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
```

Run it:
```bash
docker-compose up -d
```
Then open `http://localhost:8080` in your browser. For more detailed instructions on Docker networking and configuration, please check the [DOCKER.md](DOCKER.md) guide.

## 💻 Manual Build & Run

If you prefer to run it natively without Docker, ensure you have **Go 1.23+** and **Node.js 22+** installed.

### 1. Build the Frontend
```bash
cd frontend
npm install
npm run build
cd ..
```

### 2. Build the Backend
```bash
go build -o importer .
```

### 3. Run
```bash
./importer
```
Access the application at `http://localhost:8080`.

## ⚙️ Configuration

1. Visit the **Settings** tab in the UI.
2. Enter your **Firefly III URL** (e.g., `http://192.168.1.100:8080`).
3. Enter your **Personal Access Token** generated from Firefly III (`Options > Profile > OAuth > Personal Access Tokens`).
4. (Optional) Load default China presets for common merchants like Meituan, Ele.me, Taobao, etc.
5. Save the configuration and begin importing!

## 📖 Usage Guide

### 1. Export Billing Data
- **Alipay**: Go to the Alipay App/Website, navigate to your billing history, and export your transactions as a `.csv` file.
- **WeChat Pay**: Open WeChat, go to Me > Services > Wallet > Bills > Common Questions, and request to export your billing history as a `.xlsx` file. (The password for the file will be sent to your WeChat).

### 2. Upload & Preview
- Simply drag and drop the exported `.csv` or `.xlsx` file into the upload zone in the **Import** tab.
- The importer will automatically detect the source (Alipay or WeChat Pay) and parse the raw data into a readable transaction table.

### 3. Rules & Categories Setup
- In the **Settings** tab, you can set up advanced mapping rules.
- **Rule Groups**: Group multiple conditions (e.g., Match 'Meituan' OR 'Ele.me' in Counterparty).
- **Target Actions**: Assign the transaction to a specific Category, set default Asset/Opposing accounts, or change the transaction type (e.g., mark it as a Deposit or Transfer).
- **Test Your Rules**: Go back to the **Import** tab and preview the parsed data. The UI will instantly reflect which rules were matched and how the transactions will be assigned.

### 4. Execute Import
- Once satisfied with the preview, click **Start Import**.
- The app will communicate with Firefly III in real-time, displaying a progress bar and per-transaction status.
- **Deduplication**: If enabled in Settings, the app will check the Firefly III server for existing `external_id`s and automatically skip duplicates, so you don't have to worry about importing the same file twice.

## 📜 License

This project is licensed under the highly permissive **MIT License**. See the [LICENSE.md](LICENSE.md) file for details.
