[English](README.md)

# WG-Panel

WG-Panel 是一個使用者友好的 WireGuard 網頁管理面板，旨在簡化您的 VPN 伺服器的設定和管理。它具有清晰的層級結構（Interfaces > Servers > Clients）和強大的網路功能，包括動態 IP 支援和進階 NAT 設定。

## 快速入門

請依照以下步驟啟動並執行您的 WG-Panel 伺服器。

### 1. 初始設定

首先，執行 WG-Panel 執行檔以產生初始設定檔。

```bash
./wg-panel
```

此命令將在同一個目錄中建立一個 `config.json` 檔案，其中包含一個隨機產生的密碼，該密碼將會顯示在主控台上。

### 2. 設定

接下來，開啟 `config.json` 並編輯以下欄位以符合您的環境：

* `ListenIP`：網頁面板監聽的 IP 位址。預設為 `::`（所有介面）。
* `ListenPort`：網頁面板的連接埠。預設為 `5000`。

您也可以在此處變更 `User` 和 `Password`。如果您變更密碼，請確保使用 bcrypt 雜湊值。您可以透過執行以下命令來設定新的純文字密碼：

```bash
./wg-panel -p "your_new_password"
```

### 3. 啟動伺服器

設定完成後，再次啟動伺服器：

```bash
./wg-panel -c ./config.json
```

現在您可以透過在瀏覽器中導覽至 `http://<ListenIP>:<ListenPort>` 來存取網頁面板。

## 使用方法

WG-Panel 將您的 WireGuard 設定組織成三個層級：**Interfaces**、**Servers** 和 **Clients**。

### 1. Create an Interface

Interface 代表一個實體 WireGuard 網路裝置（例如 `wg0`）。

1. 導覽至 "Interfaces" 部分並點擊 "Create"。
2. **Name**：Interface 的簡短名稱（例如 `home-vpn`）。實際的系統介面將命名為 `wg-home-vpn`。
3. **Endpoint**：您伺服器的公開網域或 IP 位址。這將用作產生的用戶端設定檔中的端點。
4. **Private Key**：將此留空以自動產生安全的私鑰，或提供您自己的私鑰。
5. 儲存 Interface。預設情況下，它將被建立但處於停用狀態。請從主介面列表中啟用它。

### 2. Create a Server

Server 定義了 Interface 內用戶端的邏輯群組及其相關的網路設定。

1. 選擇您新建立的 Interface，然後前往 "Servers" 標籤。
2. 點擊 "Create" 並填寫伺服器詳細資訊：
 * **Server Name**：一個描述性的名稱（例如 `Personal-Devices`）。
 * **DNS**：要推送給用戶端的 DNS 伺服器（例如 `1.1.1.1`）。
 * **Enable IPv4/v6 Subnet**：為您要支援的每個 IP 系列勾選此方塊。
 * **IP Network**：此伺服器的內部網路，格式為 CIDR（例如 `10.0.0.1/24`）。伺服器將使用指定的 IP，並從該子網路中為用戶端分配位址。
 * **Routed Networks**：用戶端允許透過 VPN 存取的網路列表（CIDR 格式）。這對應於用戶端設定中的 `AllowedIPs` 設定。
 * **Block Non-Routed Network Packets**：如果啟用，此選項會產生防火牆規則，以確保用戶端*僅*能存取 **Routed Networks** 中指定的網路。來自用戶端的所有其他流量將被丟棄。

### 3. Advanced Server Features

#### Pseudo-Bridge

此功能使 VPN 用戶端看起來像與伺服器在同一個第二層網路上。它的運作方式是在指定的主介面上回應 ARP（IPv4）和鄰居發現（IPv6）請求，有效地橋接 VPN 和本地網路。

#### SNAT (Source Network Address Translation)

SNAT 允許用戶端使用伺服器的公開 IP 位址存取網際網路。其行為由 **SNAT IP/Net** 欄位決定：

* **MASQUERADE Mode**：將 **SNAT IP/Net** 留空。防火牆將對所有流量使用傳出介面的主要 IP。這是最簡單的模式。
* **SNAT Mode**：輸入單一 IP 位址。防火牆將使用此特定 IP 作為所有傳出用戶端流量的來源。
* **NETMAP Mode (僅限 IPv6)**：輸入 CIDR 格式的 IPv6 網路。這會將內部 VPN 子網路對應到公開的 IPv6 子網路。SNAT 網路的遮罩長度必須與伺服器內部 IPv6 網路的遮罩長度相符。

#### SNAT Roaming (動態 IP 支援)

SNAT Roaming 是針對具有動態公開 IP 的伺服器的一項強大功能。當指定的 **SNAT Roaming Master Interface** 的 IP 位址變更時，它會自動更新防火牆規則。此功能僅與 **SNAT** 和 **NETMAP** 模式相容。

* **在 SNAT Mode下**：
 * 將 **SNAT IP/Net** 設定為 `0.0.0.0`（對於 IPv4）或 `::`（對於 IPv6）。
 * 服務將自動偵測主介面的目前 IP，並將其用於 SNAT 規則。

* **在 NETMAP Mode 下 (IPv6)**：
 * **SNAT IP/Net** 欄位被視為*網路偏移量*。服務將此偏移量與主介面的網路位址結合，為您的 VPN 用戶端建立一個可公開路由的子網路。
 * **範例**：
 * 您伺服器的主介面具有動態 IPv6 位址：`2a0d:3a87::123/64`。其網路為 `2a0d:3a87::/64`。
 * 您的內部 WireGuard 伺服器網路為 `fd28:f50:55c2::/112`。
 * 您將 **SNAT IP/Net** 設定為 `::980d:0/112`。
 * 服務會結合主網路和偏移量，將您的內部網路 `fd28:f50:55c2::/112` 對應到公開網路 `2a0d:3a87::980d:0/112`。
 * 這使您的 VPN 用戶端即使在伺服器的公開 IPv6 前置詞變更時，也能擁有可公開路由的 IPv6 位址。

#### SNAT NETMAP pseudo-bridge

此選項將 **Pseudo-Bridge** 功能擴展到由 **SNAT NETMAP** 建立的公開子網路。它將回應對應的公開 IP 的 ARP/ND 請求，使用戶端看起來像直接在公開網路上。

### 4. Create a Client

最後，為您的伺服器建立用戶端。

1. 選擇一個 Server，然後前往 "Clients" 標籤。
2. 點擊 "Create" 並設定用戶端：
 * **IP/IPv6**：您可以手動分配特定 IP，或將其設定為 `auto`，讓 WG-Panel 從伺服器網路中分配下一個可用的位址。
 * **Private Key**：將此留空以自動產生新的金鑰對，或提供用戶端現有的私鑰。

### 5. View Client Details and Config

建立用戶端後，您可以從用戶端列表中管理它。

1. 點擊用戶端項目旁邊的展開/詳細資訊按鈕。
2. 將出現一個詳細檢視，顯示完整的 **WireGuard configuration**。
3. 點擊 **QR Code** 按鈕以顯示可用於掃描以便輕鬆匯入到行動裝置的代碼。
4. 此檢視還會顯示用戶端的即時連線狀態，包括：
 * **Data Transferred** (Upload/Download)
 * **Last Handshake** Time
 * **Endpoint** (用戶端的公開 IP 位址)