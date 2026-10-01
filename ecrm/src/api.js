const API_URL = import.meta.env.VITE_API_URL ?? '/api/v1'
const storageKey = 'ecrm.session'

let session = JSON.parse(localStorage.getItem(storageKey) || 'null')

function saveSession(value) {
  session = value
  if (value) localStorage.setItem(storageKey, JSON.stringify(value))
  else localStorage.removeItem(storageKey)
}

async function rawRequest(path, options = {}, retry = true) {
  const headers = new Headers(options.headers)
  if (options.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  if (session?.accessToken) headers.set('Authorization', `Bearer ${session.accessToken}`)
  const response = await fetch(`${API_URL}${path}`, { ...options, headers })

  if (response.status === 401 && retry && session?.refreshToken) {
    const refreshed = await fetch(`${API_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: session.refreshToken }),
    })
    if (refreshed.ok) {
      saveSession(await refreshed.json())
      return rawRequest(path, options, false)
    }
    saveSession(null)
  }

  if (!response.ok) {
    const payload = await response.json().catch(() => ({}))
    throw new Error(payload.error?.message || `HTTP ${response.status}`)
  }
  if (response.status === 204) return null
  return response.json()
}

const request = async (path, options) => (await rawRequest(path, options))?.data
const list = async (path) => (await rawRequest(path))?.data ?? []

export const api = {
  hasSession: () => Boolean(session?.accessToken),
  currentUser: () => session?.user ?? null,
  async login(identity, password) {
    const value = await rawRequest('/auth/login', { method: 'POST', body: JSON.stringify({ identity, password }) })
    saveSession(value)
    return value.user
  },
  async register(name, email, phone, password) {
    const value = await rawRequest('/auth/register', { method: 'POST', body: JSON.stringify({ name, email, phone, password }) })
    saveSession(value)
    return value.user
  },
  async logout() {
    const refreshToken = session?.refreshToken
    saveSession(null)
    if (refreshToken) await fetch(`${API_URL}/auth/logout`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refreshToken }) }).catch(() => {})
  },
  me: () => request('/auth/me'),
  customers: () => list('/customers?limit=100'),
  orders: () => list('/orders?limit=100'),
  esims: () => list('/esims?limit=100'),
  tariffs: () => list('/tariffs?limit=100'),
  suppliers: () => list('/suppliers?limit=100'),
  employees: () => list('/employees?limit=100'),
  notifications: () => list('/notifications?limit=100'),
  audit: () => list('/audit?limit=100'),
  dashboard: () => request('/dashboard'),
  createOrder: (payload) => request('/orders', { method: 'POST', body: JSON.stringify(payload) }),
  syncSuppliers: () => request('/suppliers/sync-all', { method: 'POST' }),
  syncZesimo: () => request('/integrations/zesimo/sync', { method: 'POST' }),
  zesimoStatus: () => request('/integrations/zesimo'),
}

const localDate = (value) => value ? new Date(value).toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' }) : '-'
const dataLabel = (mb) => mb >= 1024 ? `${Math.round((mb / 1024) * 10) / 10} GB` : `${mb} MB`
const orderStatuses = { created: 'Created', pending_payment: 'Pending Payment', paid: 'Paid', esim_assigned: 'eSIM Assigned', activated: 'Activated', completed: 'Completed', cancelled: 'Cancelled', refunded: 'Refunded' }
const paymentStatuses = { pending: 'Ожидает', paid: 'Оплачен', refunded: 'Возврат', failed: 'Ошибка' }
const entityLabels = { order: 'Заказ', customer: 'Клиент', esim: 'eSIM', tariff: 'Тариф', supplier: 'Поставщик', employee: 'Сотрудник' }

export async function loadCRMData() {
  const safe = (promise, fallback = []) => promise.catch(() => fallback)
  const [rawCustomers, rawOrders, rawEsims, rawTariffs, rawSuppliers, rawEmployees, rawNotifications, rawAudit, dashboard] = await Promise.all([
    api.customers(), api.orders(), api.esims(), api.tariffs(), api.suppliers(), safe(api.employees()), api.notifications(), safe(api.audit()), api.dashboard(),
  ])

  const customers = rawCustomers.map((item) => ({
    ...item, _id: item.id, id: item.code, registered: localDate(item.createdAt), last: localDate(item.lastOrderAt),
    status: item.status === 'vip' ? 'VIP' : item.status === 'active' ? 'Активный' : 'Неактивный',
  }))
  const orders = rawOrders.map((item) => ({
    ...item, _id: item.id, id: `#${item.number}`, created: localDate(item.createdAt), customer: item.customerName,
    phone: item.customerPhone, email: item.customerEmail, provider: item.supplierName, plan: item.planName,
    iccid: item.iccid || '-', payment: item.paymentMethod, paymentStatus: paymentStatuses[item.paymentStatus] || item.paymentStatus,
    status: orderStatuses[item.status] || item.status, operator: item.operatorName || '-', activation: localDate(item.activationAt), expires: localDate(item.expiresAt),
    installUrl: item.installUrl, iosTapLink: item.iosTapLink, androidTapLink: item.androidTapLink,
  }))
  const esims = rawEsims.map((item) => ({
    ...item, _id: item.id, id: item.code, provider: item.supplierName, plan: item.tariffName,
    traffic: dataLabel(item.trafficRemainingMb), dates: item.activatedAt ? `${localDate(item.activatedAt)} — ${localDate(item.expiresAt)}` : 'Не активирована',
    status: item.status[0].toUpperCase() + item.status.slice(1), customer: item.customerName || '-', order: item.orderNumber ? `#${item.orderNumber}` : '-', profit: 0,
  }))
  const tariffs = rawTariffs.map((item) => ({
    ...item, _id: item.id, id: item.code, provider: item.supplierName, gb: dataLabel(item.dataMb), days: `${item.validityDays} дней`,
    min: item.minPrice, recommended: item.recommendedPrice, status: item.status === 'active' ? (item.stock < 15 ? 'Мало остатка' : 'Активен') : 'Неактивен',
  }))
  const suppliers = rawSuppliers.map((item) => ({
    ...item, _id: item.id, id: item.code, contact: item.contactName, api: item.apiStatus === 'online' ? 'Online' : item.apiStatus === 'warning' ? 'Warning' : 'Offline',
    status: item.status === 'low_balance' ? 'Низкий баланс' : item.status === 'active' ? 'Работает' : 'Отключен',
  }))
  const employees = rawEmployees.map((item) => ({
    ...item, _id: item.id, id: item.code, role: item.role[0].toUpperCase() + item.role.slice(1),
    status: item.status === 'active' ? 'Online' : 'Disabled', access: item.permissions.includes('*') ? 'Полный доступ' : item.permissions.join(', '),
  }))
  const notifications = rawNotifications.map((item) => ({ type: item.type, text: item.message, level: item.level, id: item.id, createdAt: item.createdAt }))
  const activities = rawAudit.map((item) => ({
    user: item.actorName, action: `${item.action}: ${entityLabels[item.entityType] || item.entityType}`,
    oldValue: item.oldValue ? JSON.stringify(item.oldValue) : '-', newValue: item.newValue ? JSON.stringify(item.newValue) : '-', time: localDate(item.createdAt),
  }))
  const salesTrend = dashboard.sales.map((item) => ({ ...item, date: item.date.slice(5).split('-').reverse().join('.') }))
  const countries = dashboard.countries.map((item, index) => ({ name: item.name, value: Math.round(item.share), fill: ['#0f766e', '#2563eb', '#e95d2a', '#7c3aed', '#16a34a'][index % 5] }))
  const supplierBars = dashboard.suppliers.map((item) => ({ name: item.name, sold: item.sold, stock: item.stock }))
  return { customers, orders, esims, tariffs, suppliers, employees, notifications, activities, salesTrend, countries, supplierBars, dashboard }
}
