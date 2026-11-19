import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';

import '../core/api.dart';
import '../core/models.dart';
import 'app_state.dart';

/// 1 对 1 WebRTC 通话：信令复用 IM 长连接（CALL_*），媒体 P2P。
class CallState extends ChangeNotifier {
  final AppState app;
  final localRenderer = RTCVideoRenderer();
  final remoteRenderer = RTCVideoRenderer();

  String state = 'idle'; // idle | calling | ringing | connecting | active | ended
  String callId = '';
  String convId = '';
  String peerUid = '';
  String peerName = '';
  String callType = 'audio';
  String direction = 'outgoing';
  bool muted = false;
  bool camOn = true;
  int duration = 0;
  String notice = '';

  RTCPeerConnection? _pc;
  MediaStream? _local;
  MediaStream? _remote;
  String _pendingOffer = '';
  Timer? _timer;
  StreamSubscription? _sub;
  bool _renderersReady = false;

  CallState(this.app) {
    _sub = app.callSignals.listen(_handleSignal);
  }

  Future<void> initRenderers() async {
    if (_renderersReady) return;
    await localRenderer.initialize();
    await remoteRenderer.initialize();
    _renderersReady = true;
  }

  bool get busy => state != 'idle';

  Future<void> startCall(Conv conv, String type) async {
    if (busy) return;
    if (conv.peerId.isEmpty) {
      notice = '缺少对方信息，无法发起通话';
      notifyListeners();
      return;
    }
    _reset();
    convId = conv.convId;
    peerUid = conv.peerId;
    peerName = conv.title;
    callType = type == 'video' ? 'video' : 'audio';
    direction = 'outgoing';
    callId = 'call_${DateTime.now().millisecondsSinceEpoch}';
    state = 'calling';
    notice = '正在呼叫…';
    notifyListeners();
    try {
      await _openMedia(callType);
      _pc = await createPeerConnection(await _iceConfig());
      _bindPc();
      final offer = await _pc!.createOffer();
      await _pc!.setLocalDescription(offer);
      await _waitIce();
      app.ws.send('CALL_INVITE', {
        'callId': callId,
        'convId': convId,
        'toUid': peerUid,
        'callType': callType,
        'sdp': offer.sdp,
      });
    } catch (e) {
      _finish('呼叫失败：$e');
    }
  }

  Future<void> accept() async {
    if (state != 'ringing') return;
    state = 'connecting';
    notice = '正在连接…';
    notifyListeners();
    try {
      await _openMedia(callType);
      _pc = await createPeerConnection(await _iceConfig());
      _bindPc();
      if (_pendingOffer.isNotEmpty) {
        await _pc!.setRemoteDescription(RTCSessionDescription(_pendingOffer, 'offer'));
      }
      final answer = await _pc!.createAnswer();
      await _pc!.setLocalDescription(answer);
      await _waitIce();
      app.ws.send('CALL_ANSWER', {
        'callId': callId,
        'toUid': peerUid,
        'accept': true,
        'sdp': answer.sdp,
      });
    } catch (e) {
      app.ws.send('CALL_ANSWER', {'callId': callId, 'toUid': peerUid, 'accept': false});
      _finish('接听失败：$e');
    }
  }

  void reject() {
    if (callId.isNotEmpty && peerUid.isNotEmpty) {
      app.ws.send('CALL_ANSWER', {'callId': callId, 'toUid': peerUid, 'accept': false});
    }
    _finish('已拒绝');
  }

  void hangup() {
    if (callId.isNotEmpty && peerUid.isNotEmpty) {
      app.ws.send('CALL_HANGUP', {'callId': callId, 'toUid': peerUid});
    }
    _finish('通话已结束');
  }

  void toggleMute() {
    final stream = _local;
    if (stream == null) return;
    muted = !muted;
    for (final t in stream.getAudioTracks()) {
      t.enabled = !muted;
    }
    notifyListeners();
  }

  void toggleCam() {
    final stream = _local;
    if (stream == null) return;
    final tracks = stream.getVideoTracks();
    if (tracks.isEmpty) return;
    camOn = !camOn;
    for (final t in tracks) {
      t.enabled = camOn;
    }
    notifyListeners();
  }

  Future<void> switchCamera() async {
    final stream = _local;
    if (callType != 'video' || stream == null) return;
    final tracks = stream.getVideoTracks();
    if (tracks.isEmpty) return;
    try {
      await Helper.switchCamera(tracks.first);
    } catch (_) {}
  }

  Future<Map<String, dynamic>> _iceConfig() async {
    try {
      final d = await Api.get('/rtc/config');
      final list = d is Map ? d['iceServers'] : null;
      if (list is List && list.isNotEmpty) {
        return {
          'iceServers': list,
          'sdpSemantics': 'unified-plan',
        };
      }
    } catch (_) {}
    return {
      'iceServers': [
        {
          'urls': ['stun:stun.l.google.com:19302', 'stun:stun1.l.google.com:19302']
        }
      ],
      'sdpSemantics': 'unified-plan',
    };
  }

  Future<void> _openMedia(String type) async {
    _local = await navigator.mediaDevices.getUserMedia({
      'audio': {
        'echoCancellation': true,
        'noiseSuppression': true,
      },
      'video': type == 'video'
          ? {
              'width': {'ideal': 1280},
              'height': {'ideal': 720},
              'facingMode': 'user',
            }
          : false,
    });
    await initRenderers();
    localRenderer.srcObject = _local;
  }

  void _bindPc() {
    final pc = _pc!;
    for (final t in _local?.getTracks() ?? <MediaStreamTrack>[]) {
      pc.addTrack(t, _local!);
    }
    _remote = null;
    remoteRenderer.srcObject = null;
    pc.onTrack = (event) {
      if (event.streams.isNotEmpty) {
        _remote = event.streams.first;
        remoteRenderer.srcObject = _remote;
        if (state == 'connecting' || state == 'calling') _startTimer();
        notifyListeners();
      }
    };
    pc.onIceCandidate = (c) {
      final map = c.toMap();
      app.ws.send('CALL_ICE', {
        'callId': callId,
        'toUid': peerUid,
        'candidate': map,
      });
    };
    pc.onConnectionState = (s) {
      if (s == RTCPeerConnectionState.RTCPeerConnectionStateConnected) {
        _startTimer();
      } else if (s == RTCPeerConnectionState.RTCPeerConnectionStateFailed) {
        _finish('连接失败，可能需要 TURN 中继');
      }
    };
  }

  Future<void> _waitIce() async {
    final pc = _pc;
    if (pc == null) return;
    final done = Completer<void>();
    Timer? timer;
    void finish() {
      if (!done.isCompleted) done.complete();
      timer?.cancel();
    }

    pc.onIceGatheringState = (s) {
      if (s == RTCIceGatheringState.RTCIceGatheringStateComplete) finish();
    };
    timer = Timer(const Duration(milliseconds: 1500), finish);
    await done.future;
  }

  void _handleSignal(Map<String, dynamic> f) {
    final cmd = '${f['cmd'] ?? ''}';
    final p = f['payload'] is Map ? Map<String, dynamic>.from(f['payload']) : <String, dynamic>{};
    switch (cmd) {
      case 'CALL_INVITE_PUSH':
        if (busy) {
          app.ws.send('CALL_ANSWER', {'callId': p['callId'], 'toUid': p['fromUid'], 'accept': false});
          return;
        }
        _reset();
        callId = '${p['callId'] ?? ''}';
        peerUid = '${p['fromUid'] ?? ''}';
        peerName = '${p['fromName'] ?? p['fromUid'] ?? ''}';
        convId = '${p['convId'] ?? ''}';
        if (convId.isEmpty) convId = app.ensureConv(peerUid, peerName);
        callType = p['callType'] == 'video' ? 'video' : 'audio';
        direction = 'incoming';
        _pendingOffer = '${p['sdp'] ?? ''}';
        state = 'ringing';
        notice = callType == 'video' ? '邀请你视频通话' : '邀请你语音通话';
        notifyListeners();
        break;
      case 'CALL_ANSWER_PUSH':
        if (p['callId'] != callId) return;
        if (p['accept'] != true) {
          _finish('对方已拒绝');
          return;
        }
        state = 'connecting';
        notice = '正在连接…';
        notifyListeners();
        final sdp = '${p['sdp'] ?? ''}';
        if (sdp.isNotEmpty && _pc != null) {
          _pc!.setRemoteDescription(RTCSessionDescription(sdp, 'answer')).catchError((_) {
            _finish('连接建立失败');
          });
        }
        break;
      case 'CALL_ICE_PUSH':
        if (p['callId'] != callId || _pc == null || p['candidate'] == null) return;
        final c = Map<String, dynamic>.from(p['candidate']);
        try {
          _pc!.addCandidate(RTCIceCandidate(
            '${c['candidate'] ?? ''}',
            c['sdpMid'] == null ? null : '${c['sdpMid']}',
            (c['sdpMLineIndex'] as num?)?.toInt() ?? 0,
          ));
        } catch (_) {}
        break;
      case 'CALL_HANGUP_PUSH':
        if (p['callId'] != callId) return;
        _finish(direction == 'incoming' ? '对方已取消' : '对方已挂断');
        break;
      case 'CALL_FORCE_END_PUSH':
        if (p['callId'] != null && p['callId'] != callId) return;
        _finish('${p['reason'] ?? '管理员已结束通话'}');
        break;
      case 'CALL_ACK':
        if (f['code'] != 0 && f['code'] != null) _finish('${f['message'] ?? '呼叫失败'}');
        break;
    }
  }

  void _startTimer() {
    if (_timer != null) return;
    state = 'active';
    notice = '通话中';
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {
      duration++;
      notifyListeners();
    });
    notifyListeners();
  }

  void _finish(String text) {
    state = 'ended';
    notice = text;
    _timer?.cancel();
    _timer = null;
    notifyListeners();
    Timer(const Duration(milliseconds: 900), () {
      if (state == 'ended') {
        _reset();
        notifyListeners();
      }
    });
  }

  void _reset() {
    _timer?.cancel();
    _timer = null;
    try {
      _pc?.close();
    } catch (_) {}
    _pc = null;
    for (final t in _local?.getTracks() ?? <MediaStreamTrack>[]) {
      t.stop();
    }
    _local = null;
    _remote = null;
    localRenderer.srcObject = null;
    remoteRenderer.srcObject = null;
    state = 'idle';
    callId = '';
    convId = '';
    peerUid = '';
    peerName = '';
    callType = 'audio';
    direction = 'outgoing';
    muted = false;
    camOn = true;
    duration = 0;
    notice = '';
    _pendingOffer = '';
  }

  @override
  void dispose() {
    _sub?.cancel();
    _reset();
    localRenderer.dispose();
    remoteRenderer.dispose();
    super.dispose();
  }
}
