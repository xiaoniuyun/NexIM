# NexIM 移动端（Flutter）

基于 Flutter 的 IM 移动客户端，一套 Dart 代码同时构建 Android 与 iOS，协议层独立实现并与服务端 `/api/v1` + `/ws` 对齐。

## 技术栈与目录

- Flutter SDK：`^3.14.0`（Dart 3），状态管理 `provider ^6.1.5+1`
- 网络：`http`、`web_socket_channel`、`shared_preferences`
- 音视频：`flutter_webrtc`、`record`、`audioplayers`、`video_player`；图片/文件：`image_picker`、`file_picker`、`cached_network_image`

| 路径 | 说明 |
| --- | --- |
| `lib/main.dart` | 入口，通过 `ChangeNotifierProvider` 注入全局状态 |
| `lib/app/app_state.dart` / `call_state.dart` | 应用状态与通话状态（`ChangeNotifier`） |
| `lib/core/api.dart` | REST 客户端，默认 `http://127.0.0.1:8080/api/v1` |
| `lib/core/ws_client.dart` | WebSocket 客户端 |
| `lib/core/store.dart` / `models.dart` / `util.dart` / `image_cache.dart` | 本地缓存、数据模型、工具、图片缓存 |
| `lib/pages/` | 登录、会话列表、聊天（`chat_page.dart`）、通讯录、通话等页面 |
| `lib/widgets/avatar.dart` | 头像组件 |
| `android/` | Android Gradle Kotlin DSL 工程（`build.gradle.kts`、`settings.gradle.kts`） |
| `ios/Runner/` | iOS 工程（`AppDelegate.swift`、`Info.plist`、`Assets.xcassets`、`Runner.xcodeproj`） |
| `test/` | Flutter 测试 |

## 连接配置与协议

- REST 基地址：`lib/core/api.dart` 中的 `Api.base`，默认 `http://127.0.0.1:8080/api/v1`；真机调试需改为宿主机可达地址或部署地址。
- WebSocket：由 REST 基地址推导，端口加 1（默认 `ws://127.0.0.1:8081/ws`），见 `lib/core/api.dart`。
- 统一响应：`{code, message, data, traceId, serverTime}`，`code=0` 成功，`code=2001` 触发刷新；各端 API 层均实现 2001 自动刷新。
- WS 命令字与 Web 端一致：`AUTH / HEARTBEAT / SEND / PUSH / SYNC / READ / REVOKE / TYPING / KICK / NOTICE_PUSH`；通话信令 `CALL_INVITE / ANSWER / ICE / HANGUP` 及对应 PUSH 帧。

## 本地开发

```bash
cd d:/workcode/NexIM/mobile
flutter pub get
flutter run                     # 默认连接 127.0.0.1:8080，请确认后端已启动且地址可达
flutter test                    # 运行测试
```

Android 真机/模拟器调试时 `127.0.0.1` 指向设备自身，需将 `Api.base` 改为宿主机局域网 IP（并保证后端监听 `0.0.0.0`）。

## 构建

```bash
flutter build apk --release                         # Android APK
flutter build appbundle --release                   # Android App Bundle
flutter build ios --release --no-codesign           # iOS（签名需在 Xcode / CI 配置）
```

Android 签名与 iOS 分发证书需在各自平台工程中配置，当前仓库未内置签名材料。

## 实现边界说明

- 移动端为 **Flutter 单工程**，与需求/架构文档中「Android Kotlin + Jetpack Compose / iOS Swift + SwiftUI 双原生栈」的规划形态不同；原生平台目录仅承担打包与原生能力桥接。
- 桌面端（`desktop/`）业务层自持于 `desktop/src/shared`；移动端为独立 Flutter 实现，不共享 Web 源码，仅共享后端协议语义。
