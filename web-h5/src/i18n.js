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
    switchLang: 'EN', empty: '暂无数据', sendSuccess: '发送成功', copied: '已复制'
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
    switchLang: '中文', empty: 'No data', sendSuccess: 'Sent', copied: 'Copied'
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
