# LMDB GUI Client

![CI](https://github.com/zshimonz/lmdb-gui-client/actions/workflows/ci.yml/badge.svg)

这是一个使用 [Fyne](https://fyne.io/) 框架编写的 LMDB 图形化客户端应用。用户可以通过该客户端管理 LMDB
数据库，包括连接管理、键值对查看和编辑等功能。
目前只支持DBI为root，并且不支持分页。开启自动刷新后，无法使用value栏里的功能。

## 功能

- 连接管理：添加、编辑、删除数据库连接。
- 键值对管理：查看、添加、编辑、删除键值对。
- 自动刷新：可以设置5s自动刷新键值对。
- JSON 格式化：如果值是 JSON 格式，会自动格式化显示。

## 安装和运行

1. 克隆项目：

    ```bash
    git clone https://github.com/zshimonz/lmdb-gui-client.git
    cd lmdb-gui-client
    ```

2. 安装依赖：

   确保你已经安装了 Go 编译器，然后运行以下命令安装依赖：

    ```bash
    go mod tidy
    ```

3. 运行项目：

    ```bash
    go run main.go
    ```

4. 打包应用：
   安装fyne cmd
   ```bash
   go install fyne.io/fyne/v2/cmd/fyne@latest
   ```

    ```bash
    fyne package -os darwin -icon icon.png
    fyne package -os linux -icon icon.png
    fyne package -os windows -icon icon.png
    ```

   跨平台打包

   需要先安装
   fyne-cross https://github.com/fyne-io/fyne-cross

    ```bash
    fyne-cross windows -arch amd64 -icon icon.png --app-id com.github.zshimonz.lmdb-gui-client
    fyne-cross linux -arch amd64 -icon icon.png --app-id com.github.zshimonz.lmdb-gui-client
    fyne-cross darwin -arch arm64 -icon icon.png --app-id com.github.zshimonz.lmdb-gui-client
    ```

## 使用说明

### 主界面

主界面分为三个区域：

- **左侧栏**：管理数据库连接。
- **主窗口**：显示选中连接的键值对。
- **右侧栏**：显示选中键的值，并提供编辑和删除功能。

### 添加连接

点击左侧栏顶部的 "New Connection" 按钮，输入连接名称和数据库路径，然后点击 "Save" 保存连接。

### 编辑连接

点击连接列表中的 "Edit" 按钮，修改连接信息后点击 "Save" 保存修改。

### 删除连接

点击连接列表中的 "Delete" 按钮，删除连接。

### 查看键值对

选中一个连接后，主窗口会显示该连接中的所有键值对。可以通过输入键前缀进行过滤。

### 添加键值对

点击主窗口顶部的 "New Key" 按钮，输入键和值，然后点击 "Save" 保存键值对。

### 编辑键值对

选中一个键后，右侧栏会显示该键的值。修改值后点击 "Update" 按钮保存修改。

### 删除键值对

选中一个键后，右侧栏会显示该键的值。点击 "Delete" 按钮删除键值对。

### 自动刷新

选中 "Auto Refresh (5s)" 复选框后，程序会每隔 5 秒自动刷新键值对。
