import 'models.dart';

String _two(int v) => v.toString().padLeft(2, '0');

String timeText(int ms) {
  if (ms <= 0) return '';
  final d = DateTime.fromMillisecondsSinceEpoch(ms);
  final now = DateTime.now();
  final today = DateTime(now.year, now.month, now.day);
  final day = DateTime(d.year, d.month, d.day);
  final diff = today.difference(day).inDays;
  if (diff == 0) return '${_two(d.hour)}:${_two(d.minute)}';
  if (diff == 1) return '昨天';
  if (d.year == now.year) return '${_two(d.month)}-${_two(d.day)}';
  return '${d.year}-${_two(d.month)}-${_two(d.day)}';
}

String fullTime(int ms) {
  if (ms <= 0) return '';
  final d = DateTime.fromMillisecondsSinceEpoch(ms);
  return '${d.year}-${_two(d.month)}-${_two(d.day)} ${_two(d.hour)}:${_two(d.minute)}';
}

String bytes(int n) {
  if (n < 1024) return '$n B';
  if (n < 1024 * 1024) return '${(n / 1024).toStringAsFixed(1)} KB';
  if (n < 1024 * 1024 * 1024) return '${(n / 1024 / 1024).toStringAsFixed(1)} MB';
  return '${(n / 1024 / 1024 / 1024).toStringAsFixed(2)} GB';
}

String previewOf(Msg m) {
  if (m.status == 2) return '[消息已撤回]';
  switch (m.type) {
    case 'image':
      return '[图片]';
    case 'video':
      return '[视频]';
    case 'voice':
      return '[语音]';
    case 'file':
      final name = (m.extra is Map) ? '${m.extra['name'] ?? ''}' : '';
      return name.isEmpty ? '[文件]' : '[文件] $name';
    default:
      return m.content.replaceAll('\n', ' ');
  }
}

const _palette = ['#4f6ef7', '#3ba7d8', '#f2994a', '#3ba58c', '#a05ce6', '#e9834a'];

int avatarColor(String name) {
  if (name.isEmpty) return 0xff4f6ef7;
  var h = 0;
  for (final c in name.runes) {
    h += c;
  }
  return int.parse(_palette[h % _palette.length].substring(1), radix: 16) | 0xff000000;
}
