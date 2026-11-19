import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

/// 长连接客户端：心跳帧由服务端保持，客户端负责断线 3s 重连。
class WsClient {
  WebSocketChannel? _ch;
  StreamSubscription? _sub;
  Timer? _retry;
  bool _closed = false;
  int _seq = 0;
  String _url = '';

  bool connected = false;
  void Function(bool connected)? onState;
  void Function(Map<String, dynamic> frame)? onFrame;

  void connect(String url) {
    _url = url;
    _closed = false;
    _open();
  }

  void _open() {
    if (_closed || _url.isEmpty) return;
    try {
      final ch = WebSocketChannel.connect(Uri.parse(_url));
      _ch = ch;
      _sub = ch.stream.listen(
        (data) {
          if (!connected) {
            connected = true;
            onState?.call(true);
          }
          try {
            final f = jsonDecode('$data');
            if (f is Map) onFrame?.call(Map<String, dynamic>.from(f));
          } catch (_) {}
        },
        onDone: _down,
        onError: (_) => _down(),
        cancelOnError: true,
      );
    } catch (_) {
      _down();
    }
  }

  void _down() {
    if (connected) {
      connected = false;
      onState?.call(false);
    }
    _sub?.cancel();
    _sub = null;
    _ch = null;
    if (_closed) return;
    _retry?.cancel();
    _retry = Timer(const Duration(seconds: 3), _open);
  }

  void send(String cmd, Map<String, dynamic> payload) {
    final ch = _ch;
    if (ch == null || !connected) return;
    try {
      ch.sink.add(jsonEncode({'cmd': cmd, 'seqNo': ++_seq, 'payload': payload}));
    } catch (_) {}
  }

  void close() {
    _closed = true;
    _retry?.cancel();
    _sub?.cancel();
    try {
      _ch?.sink.close();
    } catch (_) {}
    _ch = null;
    connected = false;
  }
}
