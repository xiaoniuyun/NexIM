import 'package:flutter/material.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:provider/provider.dart';

import '../app/call_state.dart';
import '../widgets/avatar.dart';

/// 全屏通话浮层：由 CallState 驱动，承载在 MaterialApp.builder 的 Stack 顶层。
class CallOverlay extends StatelessWidget {
  const CallOverlay({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<CallState>(
      builder: (context, call, _) {
        if (call.state == 'idle') return const SizedBox.shrink();
        return Positioned.fill(
          child: Material(
            color: const Color(0xFF12151C),
            child: SafeArea(child: _body(context, call)),
          ),
        );
      },
    );
  }

  Widget _body(BuildContext context, CallState call) {
    final video = call.callType == 'video';
    return Stack(
      children: [
        Positioned.fill(
          child: video
              ? RTCVideoView(
                  call.remoteRenderer,
                  objectFit: RTCVideoViewObjectFit.RTCVideoViewObjectFitCover,
                )
              : _audioBackdrop(call),
        ),
        if (video)
          Positioned(
            right: 12,
            top: 12,
            child: ClipRRect(
              borderRadius: BorderRadius.circular(10),
              child: SizedBox(
                width: 96,
                height: 136,
                child: ColoredBox(
                  color: Colors.black45,
                  child: RTCVideoView(call.localRenderer, mirror: true),
                ),
              ),
            ),
          ),
        Column(
          children: [
            const Spacer(),
            if (!video) ...[
              Avatar(name: call.peerName, size: 88),
              const SizedBox(height: 16),
            ],
            Text(call.peerName,
                style: const TextStyle(color: Colors.white, fontSize: 20, fontWeight: FontWeight.w600)),
            const SizedBox(height: 8),
            Text(
              call.state == 'active' ? _duration(call.duration) : call.notice,
              style: const TextStyle(color: Colors.white70, fontSize: 13),
            ),
            const SizedBox(height: 36),
            _controls(context, call),
            const SizedBox(height: 40),
          ],
        ),
      ],
    );
  }

  Widget _audioBackdrop(CallState call) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Spacer(),
        Icon(call.callType == 'video' ? Icons.videocam : Icons.call,
            color: Colors.white24, size: 82),
        const Spacer(flex: 2),
      ],
    );
  }

  String _duration(int sec) {
    final m = (sec ~/ 60).toString().padLeft(2, '0');
    final s = (sec % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  Widget _controls(BuildContext context, CallState call) {
    if (call.state == 'ringing') {
      return Row(
        mainAxisAlignment: MainAxisAlignment.spaceEvenly,
        children: [
          _roundButton(Icons.call_end, Colors.red, '拒绝', call.reject),
          _roundButton(Icons.call, const Color(0xFF2FA36B), '接听', call.accept),
        ],
      );
    }
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceEvenly,
      children: [
        _roundButton(call.muted ? Icons.mic_off : Icons.mic, Colors.white24, '静音', call.toggleMute),
        if (call.callType == 'video') ...[
          _roundButton(call.camOn ? Icons.videocam : Icons.videocam_off, Colors.white24, '摄像头',
              call.toggleCam),
          _roundButton(Icons.cameraswitch_outlined, Colors.white24, '切换', call.switchCamera),
        ],
        _roundButton(Icons.call_end, Colors.red, '挂断', call.hangup),
      ],
    );
  }

  Widget _roundButton(IconData icon, Color color, String label, VoidCallback onTap) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        InkWell(
          onTap: onTap,
          customBorder: const CircleBorder(),
          child: Container(
            width: 62,
            height: 62,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
            child: Icon(icon, color: Colors.white, size: 28),
          ),
        ),
        const SizedBox(height: 8),
        Text(label, style: const TextStyle(color: Colors.white70, fontSize: 12)),
      ],
    );
  }
}
