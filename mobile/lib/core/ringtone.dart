import 'dart:async';
import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';

import 'package:audioplayers/audioplayers.dart';
import 'package:flutter/services.dart';
import 'package:path_provider/path_provider.dart';

/// 通话铃声：本地合成 WAV 并循环播放，无需音频资源文件。
/// - 去电回铃音：450Hz，响 1s 停 4s
/// - 来电铃声：440+480Hz 双音，响 2s 停 2s，同时振动
class CallRingtone {
  static const int _sampleRate = 22050;

  final AudioPlayer _player = AudioPlayer();
  Timer? _vibrateTimer;
  String _mode = '';

  bool get playing => _mode.isNotEmpty;

  /// 去电回铃音（等待对方接听）
  Future<void> playOutgoing() => _start('outgoing');

  /// 来电铃声 + 振动
  Future<void> playIncoming() => _start('incoming');

  Future<void> _start(String mode) async {
    if (_mode == mode) return;
    await stop();
    _mode = mode;
    try {
      final path = await _file(mode);
      await _player.setReleaseMode(ReleaseMode.loop);
      await _player.setVolume(1);
      await _player.play(DeviceFileSource(path));
    } catch (_) {}
    if (mode == 'incoming') {
      HapticFeedback.vibrate();
      _vibrateTimer = Timer.periodic(const Duration(milliseconds: 1500), (_) {
        HapticFeedback.vibrate();
      });
    }
  }

  Future<void> stop() async {
    final was = _mode.isNotEmpty;
    _mode = '';
    _vibrateTimer?.cancel();
    _vibrateTimer = null;
    if (!was) return;
    try {
      await _player.stop();
    } catch (_) {}
  }

  Future<void> dispose() async {
    await stop();
    try {
      await _player.dispose();
    } catch (_) {}
  }

  Future<String> _file(String mode) async {
    final dir = await getTemporaryDirectory();
    final f = File('${dir.path}/nexim_ring_$mode.wav');
    if (!await f.exists() || await f.length() < 1000) {
      final bytes = mode == 'incoming'
          ? _wav([_Tone(2, 440, 480), _Tone(2)])
          : _wav([_Tone(1, 450), _Tone(4)]);
      await f.writeAsBytes(bytes, flush: true);
    }
    return f.path;
  }

  /// 生成 16bit 单声道 PCM WAV
  Uint8List _wav(List<_Tone> tones) {
    final total = tones.fold<int>(0, (s, t) => s + (t.seconds * _sampleRate).round());
    final bytes = ByteData(44 + total * 2);
    void w(int offset, String s) {
      for (var i = 0; i < s.length; i++) {
        bytes.setUint8(offset + i, s.codeUnitAt(i));
      }
    }

    w(0, 'RIFF');
    bytes.setUint32(4, 36 + total * 2, Endian.little);
    w(8, 'WAVE');
    w(12, 'fmt ');
    bytes.setUint32(16, 16, Endian.little);
    bytes.setUint16(20, 1, Endian.little); // PCM
    bytes.setUint16(22, 1, Endian.little); // 单声道
    bytes.setUint32(24, _sampleRate, Endian.little);
    bytes.setUint32(28, _sampleRate * 2, Endian.little);
    bytes.setUint16(32, 2, Endian.little);
    bytes.setUint16(34, 16, Endian.little);
    w(36, 'data');
    bytes.setUint32(40, total * 2, Endian.little);

    var offset = 0;
    for (final t in tones) {
      final n = (t.seconds * _sampleRate).round();
      for (var k = 0; k < n; k++) {
        var v = 0.0;
        if (t.freqs.isNotEmpty) {
          final tt = k / _sampleRate;
          for (final f in t.freqs) {
            v += math.sin(2 * math.pi * f * tt) / t.freqs.length;
          }
          // 10ms 淡入淡出，避免爆音
          final fade = math.min(1.0, math.min(tt / 0.01, (t.seconds - tt) / 0.01));
          v *= 0.35 * math.max(0.0, fade);
        }
        bytes.setInt16(44 + (offset + k) * 2, (v * 32767).round(), Endian.little);
      }
      offset += n;
    }
    return bytes.buffer.asUint8List();
  }
}

class _Tone {
  final double seconds;
  final List<double> freqs;

  _Tone(this.seconds, [double? a, double? b])
      : freqs = [
          if (a != null) a,
          if (b != null) b,
        ];
}
