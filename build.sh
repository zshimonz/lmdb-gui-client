#!/bin/bash

# 设置应用程序名称和图标路径
APP_NAME="LMDB GUI Client"
ICON_PATH="icon.png"
APP_ID="com.example.lmdbguiclient"

# 定义目标平台和架构
PLATFORMS=("darwin" "linux" "windows")
ARCHITECTURES=("amd64" "arm64")

# 创建构建目录
mkdir -p build

# 循环遍历平台和架构，进行编译和打包
for platform in "${PLATFORMS[@]}"; do
  for arch in "${ARCHITECTURES[@]}"; do
    echo "Packaging for $platform-$arch..."
    if [[ "$platform" == "darwin" || "$platform" == "windows" ]]; then
      fyne-cross $platform --app-id $APP_ID --icon $ICON_PATH --name "$APP_NAME" --arch $arch
    else
      fyne-cross $platform --icon $ICON_PATH --name "$APP_NAME" --arch $arch
    fi
    # 检查构建是否成功
    if [ $? -ne 0 ]; then
      echo "Error: Packaging for $platform-$arch failed"
      exit 1
    fi
    # 移动构建产物到指定目录
    mkdir -p "build/${platform}_${arch}"
    mv fyne-cross/bin/${platform}-${arch}/* build/${platform}_${arch}/
  done
done

echo "Packaging complete."
