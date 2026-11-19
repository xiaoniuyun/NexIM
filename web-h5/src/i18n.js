import { ref } from 'vue'

const DICT = {
  zh: {
    login: '登录', register: '注册', account: '账号', password: '密码', nickname: '昵称',
    loginHint: '请输入账号或手机号', pwdHint: '请输入密码', loginBtn: '登录',
    regBtn: '注册并登录', demoAccount: '演示账号', msg: '消息', contacts: '通讯录',
    discover: '发现', mine: '我的', search: '搜索', newFriends: '新的朋友',
    addFriend: '添加朋友', createGroup: '发起群聊', inputSearchUser: '输入手机号或昵称搜索',
    confidential: '手机号 / 昵称 / 部门', noConv: '暂无会话', noFriend: '暂无好友',
    online: '在线', busy: '忙碌', invisible: '隐身', send: '发送', inputMsg: '输入消息...',
    settings: '设置', devices: '已登录设备', status: '在线状态', logout: '退出登录',
    notification: '消息通知', privacy: '隐私与安全', general: '通用设置',
    groupList: '群聊', org: '组织架构', files: '文件助手', favorites: '收藏',
    discoverHint: '企业公告', back: '返回', confirm: '确定', cancel: '取消',
    switchLang: 'EN', empty: '暂无数据', sendSuccess: '发送成功', copied: '已复制',
    all: '全部', unreadOnly: '未读', groupOnly: '群聊', readAll: '全部已读',
    pin: '置顶', unpin: '取消置顶', mute: '免打扰', unmute: '取消免打扰', del: '删除',
    convActions: '会话操作', markRead: '标为已读', longPressTip: '长按会话可置顶、免打扰或删除',
    forward: '转发', favorite: '收藏', revoke: '撤回', copy: '复制', deleteLocal: '删除',
    sessionSettings: '会话设置', searchInChat: '搜索聊天记录', members: '群成员',
    invite: '邀请成员', leaveGroup: '退出群聊', pinned: '已置顶', muted: '免打扰',
    noResult: '无搜索结果', attach: '附件', image: '图片', video: '视频', file: '文件',
    forwardTo: '转发给', forwardNote: '留言（可选）', sendTo: '发送', favoritesEmpty: '还没有收藏，长按消息即可收藏',
    allRead: '已全部标记为已读', settingsSaved: '设置已保存', unblock: '移除', blacklist: '黑名单',
    addBlacklist: '拉黑', blacklistEmpty: '黑名单为空', cacheCleared: '缓存已清理',
    remoteLogout: '远程下线', deviceKicked: '设备已下线', pinSuccess: '操作成功',
    fontSmall: '小', fontNormal: '标准', fontLarge: '大', theme: '深色模式',
    notifSound: '声音提醒', notifVibrate: '震动提醒', privacyTip: '隐私设置仅作用于当前账号'
  },
  en: {
    login: 'Sign in', register: 'Register', account: 'Account', password: 'Password', nickname: 'Nickname',
    loginHint: 'Account or phone', pwdHint: 'Password', loginBtn: 'Sign in',
    regBtn: 'Register & Sign in', demoAccount: 'Demo', msg: 'Chats', contacts: 'Contacts',
    discover: 'Discover', mine: 'Me', search: 'Search', newFriends: 'New Friends',
    addFriend: 'Add Friend', createGroup: 'New Group', inputSearchUser: 'Search by phone or nickname',
    confidential: 'Phone / nickname / dept', noConv: 'No conversations', noFriend: 'No friends',
    online: 'Online', busy: 'Busy', invisible: 'Invisible', send: 'Send', inputMsg: 'Type a message...',
    settings: 'Settings', devices: 'Devices', status: 'Status', logout: 'Sign out',
    notification: 'Notifications', privacy: 'Privacy & Security', general: 'General',
    groupList: 'Groups', org: 'Organization', files: 'Files', favorites: 'Favorites',
    discoverHint: 'Announcements', back: 'Back', confirm: 'OK', cancel: 'Cancel',
    switchLang: '中文', empty: 'No data', sendSuccess: 'Sent', copied: 'Copied',
    all: 'All', unreadOnly: 'Unread', groupOnly: 'Groups', readAll: 'Read all',
    pin: 'Pin', unpin: 'Unpin', mute: 'Mute', unmute: 'Unmute', del: 'Delete',
    convActions: 'Conversation', markRead: 'Mark as read', longPressTip: 'Long press a chat to pin, mute or delete',
    forward: 'Forward', favorite: 'Favorite', revoke: 'Recall', copy: 'Copy', deleteLocal: 'Delete',
    sessionSettings: 'Chat settings', searchInChat: 'Search in chat', members: 'Members',
    invite: 'Invite', leaveGroup: 'Leave group', pinned: 'Pinned', muted: 'Muted',
    noResult: 'No results', attach: 'Attachment', image: 'Image', video: 'Video', file: 'File',
    forwardTo: 'Forward to', forwardNote: 'Note (optional)', sendTo: 'Send', favoritesEmpty: 'No favorites yet. Long press a message to save it.',
    allRead: 'All marked as read', settingsSaved: 'Saved', unblock: 'Remove', blacklist: 'Blocked users',
    addBlacklist: 'Block', blacklistEmpty: 'No blocked users', cacheCleared: 'Cache cleared',
    remoteLogout: 'Sign out device', deviceKicked: 'Device signed out', pinSuccess: 'Done',
    fontSmall: 'Small', fontNormal: 'Normal', fontLarge: 'Large', theme: 'Dark mode',
    notifSound: 'Sound', notifVibrate: 'Vibration', privacyTip: 'Applies to current account only'
  }
}

export const lang = ref(localStorage.getItem('h5_lang') || 'zh')
export function t(key) {
  const d = DICT[lang.value] || DICT.zh
  return d[key] != null ? d[key] : (DICT.zh[key] != null ? DICT.zh[key] : key)
}
export function toggleLang() {
  lang.value = lang.value === 'zh' ? 'en' : 'zh'
  localStorage.setItem('h5_lang', lang.value)
}
