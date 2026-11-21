import 'package:flutter/material.dart';

/// 服务协议 / 隐私政策
class LegalPage extends StatelessWidget {
  /// true：服务协议；false：隐私政策
  final bool service;
  const LegalPage({super.key, this.service = true});

  static const _updated = '更新日期：2026年11月21日';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFFF7F8FA),
      appBar: AppBar(
        title: Text(service ? '服务协议' : '隐私政策'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_ios_new, size: 18),
          onPressed: () => Navigator.of(context).maybePop(),
        ),
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 32),
          children: [
            Text(
              service ? 'NexIM 服务协议' : 'NexIM 隐私政策',
              style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 6),
            const Text(_updated, style: TextStyle(fontSize: 12, color: Color(0xFF9AA3B2))),
            const SizedBox(height: 18),
            ...(service ? _serviceSections : _privacySections).map(_section),
          ],
        ),
      ),
    );
  }

  Widget _section(MapEntry<String, String> e) => Padding(
        padding: const EdgeInsets.only(bottom: 18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(e.key, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
            const SizedBox(height: 7),
            Text(
              e.value,
              style: const TextStyle(fontSize: 13.5, height: 1.75, color: Color(0xFF4A5466)),
            ),
          ],
        ),
      );

  static const List<MapEntry<String, String>> _serviceSections = [
    MapEntry('一、协议的接受',
        '欢迎使用 NexIM 企业即时通讯服务（以下简称“本服务”）。您在使用本服务前，请认真阅读并充分理解本协议全部内容。您勾选同意、注册、登录或使用本服务，即表示您已接受本协议的全部约定。'),
    MapEntry('二、账号与安全',
        '1. 您应使用真实、合法的身份信息注册账号，并妥善保管账号、密码与验证码。\n'
        '2. 通过短信验证码登录即视为您本人操作，因验证码泄露导致的损失由您自行承担。\n'
        '3. 如发现账号被盗用或存在安全风险，请立即联系企业管理员或通过反馈渠道通知我们。'),
    MapEntry('三、使用规范',
        '您承诺不利用本服务从事任何违反法律法规或公序良俗的行为，包括但不限于：发布违法违规信息、侵犯他人合法权益、传播恶意程序、干扰服务正常运行、未经授权访问他人数据等。'),
    MapEntry('四、企业组织与外部联系人',
        '由企业管理员添加到组织架构的成员将自动加入企业全员群；您自行注册的账号属于外部联系人，不会自动加入企业群组，可通过邀请方式参与协作。'),
    MapEntry('五、消息与文件',
        '您在会话中发送的文字、图片、视频、语音与文件等内容的权利归您所有。为向您提供消息同步、多端漫游与文件下载服务，我们会在服务端存储上述内容。请勿在会话中传输依法禁止传播的信息。'),
    MapEntry('六、服务变更、中断与终止',
        '因系统维护、升级或不可抗力导致服务中断的，我们将尽可能提前通知。您违反本协议或相关法律法规的，我们有权视情况采取限制功能、暂停或终止服务等措施。'),
    MapEntry('七、免责声明',
        '本服务按“现状”提供。因网络环境、终端设备、第三方服务等非我方原因造成的服务异常或数据延迟，我们将尽力协助处理，但不承担由此产生的间接损失。'),
    MapEntry('八、协议更新与联系我们',
        '我们可能根据业务发展更新本协议，更新后将在应用内公布。若您继续使用本服务，即视为接受更新后的协议。如有疑问，请联系企业管理员或发送邮件至 support@nexim.local。'),
  ];

  static const List<MapEntry<String, String>> _privacySections = [
    MapEntry('一、我们收集的信息',
        '1. 账号信息：用户名、昵称、头像、部门、职务、手机号（如您提供）。\n'
        '2. 通讯信息：您发送和接收的会话消息、群组、联系人及组织架构信息。\n'
        '3. 媒体信息：您在会话中主动发送的图片、视频、语音与文件。\n'
        '4. 设备与日志：设备型号、操作系统、应用版本、登录时间、IP 地址等，用于保障账号与服务安全。'),
    MapEntry('二、我们如何使用信息',
        '上述信息仅用于：提供消息收发与多端同步、音视频通话、群组与联系人管理、账号安全校验、故障排查与服务优化。我们不会将您的个人信息用于与上述目的无关的用途。'),
    MapEntry('三、信息的存储与保护',
        '您的信息存储于企业自建的服务器中，存储期限为实现服务目的所必需的最短时间。我们采用访问控制、传输加密、密码哈希等措施保护数据安全；语音、图片与视频文件以加密通道传输。'),
    MapEntry('四、信息的共享与披露',
        '除以下情形外，我们不会向第三方共享您的个人信息：\n'
        '1. 获得您的明确同意；\n'
        '2. 根据法律法规或监管部门的强制要求；\n'
        '3. 为完成音视频通话而必需的信令与中继服务（仅传输必要的连接信息）。'),
    MapEntry('五、您的权利',
        '您可以查阅、修改昵称、头像、签名等个人资料，也可以删除本地缓存与会话记录。如需注销账号或导出、删除您的个人信息，请联系企业管理员处理。'),
    MapEntry('六、权限说明',
        '麦克风：用于语音消息录制与语音通话；\n'
        '相机与相册：用于拍摄或选择图片、视频发送；\n'
        '通知：用于接收新消息提醒；\n'
        '以上权限均在您使用对应功能时申请，您可以随时在系统设置中关闭。'),
    MapEntry('七、未成年人保护',
        '本服务面向企业办公场景。若您为未成年人，请在监护人陪同下阅读本政策并在取得同意后使用本服务。'),
    MapEntry('八、政策更新与联系我们',
        '我们可能适时更新本隐私政策，更新后将在应用内公布。如您对本政策有任何疑问、意见或投诉，请联系企业管理员或发送邮件至 privacy@nexim.local。'),
  ];
}
