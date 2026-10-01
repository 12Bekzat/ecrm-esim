import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import { api, loadCRMData } from './api'
import {
  Activity,
  AlertTriangle,
  Apple,
  BadgeCheck,
  Banknote,
  BarChart3,
  Bell,
  Calendar,
  Check,
  ChevronDown,
  CircleDollarSign,
  Copy,
  CreditCard,
  Database,
  Download,
  FileText,
  Filter,
  Globe2,
  KeyRound,
  LayoutDashboard,
  Lock,
  LogOut,
  Mail,
  Menu,
  PackageCheck,
  Phone,
  Plus,
  QrCode,
  RefreshCcw,
  Search,
  Send,
  ShieldCheck,
  ShoppingCart,
  Smartphone,
  Tags,
  Truck,
  UserPlus,
  Users,
  Wifi,
  X,
  Zap,
} from 'lucide-react'
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

const money = new Intl.NumberFormat('ru-KZ', {
  style: 'currency',
  currency: 'KZT',
  maximumFractionDigits: 0,
})

const currencyFormatters = new Map([['KZT', money]])
const formatMoney = (value, currency = 'KZT') => {
  if (!currencyFormatters.has(currency)) {
    currencyFormatters.set(currency, new Intl.NumberFormat('ru-KZ', { style: 'currency', currency, maximumFractionDigits: 2 }))
  }
  return currencyFormatters.get(currency).format(value ?? 0)
}

const number = new Intl.NumberFormat('ru-RU')

const views = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { id: 'orders', label: 'Заказы', icon: ShoppingCart },
  { id: 'new-order', label: 'Создать заказ', icon: Plus },
  { id: 'customers', label: 'Клиенты', icon: Users },
  { id: 'esims', label: 'eSIM / ICCID', icon: Smartphone },
  { id: 'tariffs', label: 'Тарифы', icon: Tags },
  { id: 'suppliers', label: 'Поставщики', icon: Truck },
  { id: 'sales', label: 'Продажи', icon: BarChart3 },
  { id: 'finance', label: 'Финансы', icon: CircleDollarSign },
  { id: 'reports', label: 'Отчеты', icon: FileText },
  { id: 'api', label: 'API поставщиков', icon: Database },
  { id: 'notifications', label: 'Уведомления', icon: Bell },
  { id: 'employees', label: 'Сотрудники', icon: ShieldCheck },
  { id: 'activity', label: 'История', icon: Activity },
]

const customers = [
  {
    id: 'C-1024',
    name: 'Бекарыс Иванов',
    phone: '+7 700 300 2405',
    email: 'ivanov@example.com',
    country: 'Казахстан',
    registered: '12.08.2026',
    orders: 4,
    total: 48600,
    profit: 14650,
    last: '27.08.2026',
    status: 'Активный',
  },
  {
    id: 'C-1025',
    name: 'Алия Ахметова',
    phone: '+7 701 555 9031',
    email: 'aliya@travel.kz',
    country: 'Казахстан',
    registered: '18.08.2026',
    orders: 3,
    total: 27700,
    profit: 9100,
    last: '27.08.2026',
    status: 'Активный',
  },
  {
    id: 'C-1026',
    name: 'Михаил Петров',
    phone: '+7 777 820 1120',
    email: 'm.petrov@mail.com',
    country: 'Россия',
    registered: '21.08.2026',
    orders: 1,
    total: 6900,
    profit: 2300,
    last: '27.08.2026',
    status: 'Активный',
  },
  {
    id: 'C-1027',
    name: 'Анна Excel Travel',
    phone: '+7 707 090 2215',
    email: 'anna@exceltravel.kz',
    country: 'Казахстан',
    registered: '05.08.2026',
    orders: 9,
    total: 112400,
    profit: 39400,
    last: '26.08.2026',
    status: 'VIP',
  },
]

const orders = [
  {
    id: '#55731',
    created: '27.08.2026 15:28',
    customer: 'Бекарыс Иванов',
    phone: '+7 700 300 2405',
    email: 'ivanov@example.com',
    country: 'Турция',
    plan: '15 GB / 15 дней',
    iccid: '89812003919131016625',
    provider: 'BC Telecom',
    cost: 3900,
    price: 6900,
    fee: 210,
    tax: 345,
    profit: 2445,
    payment: 'Kaspi QR',
    paymentStatus: 'Оплачен',
    status: 'eSIM Assigned',
    operator: 'Джон',
    activation: '27.08.2026',
    expires: '11.09.2026',
  },
  {
    id: '#55730',
    created: '27.08.2026 15:12',
    customer: 'Алия Ахметова',
    phone: '+7 701 555 9031',
    email: 'aliya@travel.kz',
    country: 'Китай',
    plan: '20 GB / 30 дней',
    iccid: '89812003919131016275',
    provider: 'Joytel',
    cost: 5600,
    price: 8900,
    fee: 267,
    tax: 445,
    profit: 2588,
    payment: 'Карта',
    paymentStatus: 'Оплачен',
    status: 'Activated',
    operator: 'Алия',
    activation: '27.08.2026',
    expires: '26.09.2026',
  },
  {
    id: '#55729',
    created: '27.08.2026 14:41',
    customer: 'Михаил Петров',
    phone: '+7 777 820 1120',
    email: 'm.petrov@mail.com',
    country: 'Европа 33',
    plan: '10 GB / 15 дней',
    iccid: '89812003919131015400',
    provider: 'Airalo Pro',
    cost: 4400,
    price: 6900,
    fee: 207,
    tax: 345,
    profit: 1948,
    payment: 'Наличные',
    paymentStatus: 'Ожидает',
    status: 'Pending Payment',
    operator: 'Турсын',
    activation: '-',
    expires: '-',
  },
  {
    id: '#55728',
    created: '27.08.2026 13:59',
    customer: 'Анна Excel Travel',
    phone: '+7 707 090 2215',
    email: 'anna@exceltravel.kz',
    country: 'Вьетнам',
    plan: '4 GB / 15 дней',
    iccid: '89859000040002822560',
    provider: 'Joytel',
    cost: 3100,
    price: 5500,
    fee: 165,
    tax: 275,
    profit: 1960,
    payment: 'Kaspi QR',
    paymentStatus: 'Оплачен',
    status: 'Completed',
    operator: 'Анна',
    activation: '26.08.2026',
    expires: '10.09.2026',
  },
]

const esims = [
  {
    id: 'ES-8916',
    iccid: '89812003919131016625',
    imsi: '401770030240511',
    country: 'Турция',
    plan: '15 GB / 15 дней',
    traffic: '12.8 GB',
    dates: '27.08 - 11.09',
    status: 'Active',
    provider: 'BC Telecom',
    cost: 3900,
    price: 6900,
    profit: 2445,
    customer: 'Бекарыс Иванов',
    order: '#55731',
  },
  {
    id: 'ES-8917',
    iccid: '89812003919131016275',
    imsi: '401770030240512',
    country: 'Китай',
    plan: '20 GB / 30 дней',
    traffic: '19.1 GB',
    dates: '27.08 - 26.09',
    status: 'Active',
    provider: 'Joytel',
    cost: 5600,
    price: 8900,
    profit: 2588,
    customer: 'Алия Ахметова',
    order: '#55730',
  },
  {
    id: 'ES-8918',
    iccid: '89812003919131015400',
    imsi: '401770030240513',
    country: 'Европа 33',
    plan: '10 GB / 15 дней',
    traffic: '10 GB',
    dates: 'Не активирована',
    status: 'Reserved',
    provider: 'Airalo Pro',
    cost: 4400,
    price: 6900,
    profit: 1948,
    customer: 'Михаил Петров',
    order: '#55729',
  },
  {
    id: 'ES-8919',
    iccid: '89812003919131014002',
    imsi: '401770030240514',
    country: 'ОАЭ',
    plan: '10 GB / 15 дней',
    traffic: '10 GB',
    dates: 'На складе',
    status: 'Available',
    provider: 'BC Telecom',
    cost: 6100,
    price: 11500,
    profit: 4950,
    customer: '-',
    order: '-',
  },
]

const tariffs = [
  {
    id: 'T-401',
    name: 'Turkey Lite',
    country: 'Турция',
    region: 'TR',
    gb: '3 GB',
    days: '7 дней',
    provider: 'BC Telecom',
    cost: 1200,
    price: 3500,
    min: 3200,
    recommended: 3900,
    stock: 86,
    status: 'Активен',
  },
  {
    id: 'T-402',
    name: 'China TikTok',
    country: 'Китай',
    region: 'CN',
    gb: '20 GB',
    days: '30 дней',
    provider: 'Joytel',
    cost: 5600,
    price: 8900,
    min: 8200,
    recommended: 9500,
    stock: 54,
    status: 'Активен',
  },
  {
    id: 'T-403',
    name: 'Europe 33',
    country: 'Европа',
    region: 'EU',
    gb: '10 GB',
    days: '15 дней',
    provider: 'Airalo Pro',
    cost: 4400,
    price: 6900,
    min: 6500,
    recommended: 7400,
    stock: 28,
    status: 'Активен',
  },
  {
    id: 'T-404',
    name: 'UAE Business',
    country: 'ОАЭ',
    region: 'GCC',
    gb: '10 GB',
    days: '15 дней',
    provider: 'BC Telecom',
    cost: 6100,
    price: 11500,
    min: 10800,
    recommended: 12400,
    stock: 12,
    status: 'Мало остатка',
  },
]

const suppliers = [
  {
    id: 'S-01',
    name: 'BC Telecom',
    contact: 'Bek Connect',
    api: 'Online',
    currency: 'KZT',
    deposit: 1800000,
    balance: 748000,
    countries: 64,
    esims: 382,
    terms: 'Предоплата, автообновление',
    status: 'Работает',
  },
  {
    id: 'S-02',
    name: 'Joytel',
    contact: 'Global SIM Desk',
    api: 'Online',
    currency: 'USD',
    deposit: 2400000,
    balance: 325000,
    countries: 82,
    esims: 291,
    terms: 'Постоплата 7 дней',
    status: 'Работает',
  },
  {
    id: 'S-03',
    name: 'Airalo Pro',
    contact: 'Partner API',
    api: 'Warning',
    currency: 'USD',
    deposit: 1200000,
    balance: 96000,
    countries: 118,
    esims: 73,
    terms: 'API only',
    status: 'Низкий баланс',
  },
]

const employees = [
  {
    id: 'E-01',
    name: 'Джон',
    phone: '+7 701 111 2233',
    email: 'aktoty@esim.kz',
    role: 'Admin',
    login: 'aktoty',
    status: 'Online',
    created: '01.08.2026',
    access: 'Полный доступ',
  },
  {
    id: 'E-02',
    name: 'Алия',
    phone: '+7 701 332 4455',
    email: 'aliya.ops@esim.kz',
    role: 'Operator',
    login: 'aliya',
    status: 'Online',
    created: '07.08.2026',
    access: 'Заказы, клиенты, eSIM',
  },
  {
    id: 'E-03',
    name: 'Турсын',
    phone: '+7 705 990 1112',
    email: 'finance@esim.kz',
    role: 'Finance',
    login: 'finance',
    status: 'Away',
    created: '09.08.2026',
    access: 'Финансы, продажи, отчеты',
  },
]

const salesTrend = [
  { date: '21.08', revenue: 188000, profit: 61600, cost: 102000 },
  { date: '22.08', revenue: 212000, profit: 74200, cost: 116500 },
  { date: '23.08', revenue: 178000, profit: 58400, cost: 99600 },
  { date: '24.08', revenue: 244000, profit: 82300, cost: 131000 },
  { date: '25.08', revenue: 291000, profit: 98600, cost: 158000 },
  { date: '26.08', revenue: 326000, profit: 117500, cost: 176000 },
  { date: '27.08', revenue: 354000, profit: 128400, cost: 189000 },
]

const countries = [
  { name: 'Турция', value: 34, fill: '#0f766e' },
  { name: 'Китай', value: 24, fill: '#2563eb' },
  { name: 'Европа', value: 18, fill: '#e95d2a' },
  { name: 'ОАЭ', value: 14, fill: '#7c3aed' },
  { name: 'Вьетнам', value: 10, fill: '#16a34a' },
]

const supplierBars = [
  { name: 'BC', sold: 184, stock: 382 },
  { name: 'Joytel', sold: 139, stock: 291 },
  { name: 'Airalo', sold: 72, stock: 73 },
  { name: 'Nomad', sold: 44, stock: 118 },
]

const activities = [
  {
    user: 'Джон',
    action: 'Создан заказ #55731',
    oldValue: '-',
    newValue: 'eSIM Assigned',
    time: '27.08.2026 15:29',
  },
  {
    user: 'Алия',
    action: 'Изменен статус eSIM ES-8917',
    oldValue: 'Sold',
    newValue: 'Active',
    time: '27.08.2026 15:14',
  },
  {
    user: 'System API',
    action: 'Обновлен баланс поставщика',
    oldValue: '101 000 KZT',
    newValue: '96 000 KZT',
    time: '27.08.2026 14:58',
  },
  {
    user: 'Турсын',
    action: 'Проверен платеж #55729',
    oldValue: 'Created',
    newValue: 'Pending Payment',
    time: '27.08.2026 14:42',
  },
]

const notifications = [
  { type: 'Баланс', text: 'Airalo Pro ниже лимита 100 000 KZT', level: 'danger' },
  { type: 'Склад', text: 'ОАЭ Business: осталось 12 eSIM', level: 'warning' },
  { type: 'API', text: 'Joytel обновил цены по Китаю', level: 'info' },
]

const CRMContext = createContext(null)
const useCRM = () => useContext(CRMContext)

function App() {
  const [authed, setAuthed] = useState(api.hasSession())
  const [user, setUser] = useState(api.currentUser())
  const [authMode, setAuthMode] = useState('login')
  const [activeView, setActiveView] = useState('dashboard')
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [selectedOrder, setSelectedOrder] = useState(null)
  const [crmData, setCRMData] = useState({ customers, orders, esims, tariffs, suppliers, employees, salesTrend, countries, supplierBars, activities, notifications })

  const refreshData = async () => {
    const value = await loadCRMData()
    setCRMData(value)
    return value
  }

  useEffect(() => {
    if (!authed) return
    refreshData().catch((error) => console.error('CRM data loading failed', error))
  }, [authed])

  const activeItem = views.find((item) => item.id === activeView) ?? views[0]

  const globalMatches = useMemo(() => {
    const value = query.trim().toLowerCase()
    if (!value) return []
    return [
      ...crmData.orders.map((item) => ({ type: 'Заказ', title: item.id, meta: `${item.customer}, ${item.iccid}`, view: 'orders' })),
      ...crmData.customers.map((item) => ({ type: 'Клиент', title: item.name, meta: `${item.phone}, ${item.email}`, view: 'customers' })),
      ...crmData.esims.map((item) => ({ type: 'eSIM', title: item.iccid, meta: `${item.customer}, ${item.status}`, view: 'esims' })),
      ...crmData.tariffs.map((item) => ({ type: 'Тариф', title: item.name, meta: `${item.country}, ${item.provider}`, view: 'tariffs' })),
      ...crmData.suppliers.map((item) => ({ type: 'Поставщик', title: item.name, meta: `${item.api}, ${item.status}`, view: 'suppliers' })),
    ].filter((item) => `${item.title} ${item.meta}`.toLowerCase().includes(value)).slice(0, 6)
  }, [query, crmData])

  if (!authed) {
    return <AuthScreen authMode={authMode} setAuthMode={setAuthMode} onEnter={(value) => { setUser(value); setAuthed(true) }} />
  }

  return (
    <CRMContext.Provider value={{ ...crmData, refreshData }}>
    <div className="app-shell">
      <aside className={`sidebar ${sidebarOpen ? 'is-open' : ''}`}>
        <div className="brand">
          <div className="brand-mark">
            <Wifi size={18} />
          </div>
          <div>
            <strong>eSIM Control</strong>
            <span>CRM Platform</span>
          </div>
        </div>

        <nav className="nav-list" aria-label="Основная навигация">
          {views.map((item) => {
            const Icon = item.icon
            return (
              <button
                className={activeView === item.id ? 'nav-item active' : 'nav-item'}
                key={item.id}
                type="button"
                onClick={() => {
                  setActiveView(item.id)
                  setSidebarOpen(false)
                }}
              >
                <Icon size={18} />
                <span>{item.label}</span>
              </button>
            )
          })}
        </nav>

        <div className="sidebar-panel">
          <div className="sidebar-panel-icon">
            <Zap size={16} />
          </div>
          <div>
            <strong>API sync</strong>
            <span>3 поставщика онлайн</span>
          </div>
        </div>
      </aside>

      <main className="workspace">
        <Topbar
          activeItem={activeItem}
          query={query}
          setQuery={setQuery}
          matches={globalMatches}
          onMenu={() => setSidebarOpen((value) => !value)}
          onNavigate={setActiveView}
          user={user}
          onLogout={async () => { await api.logout(); setUser(null); setAuthed(false) }}
        />
        <section className="content-area">
          <ViewRouter activeView={activeView} setActiveView={setActiveView} setSelectedOrder={setSelectedOrder} />
        </section>
      </main>

      {selectedOrder && <ActivationModal order={selectedOrder} onClose={() => setSelectedOrder(null)} />}
    </div>
    </CRMContext.Provider>
  )
}

function AuthScreen({ authMode, setAuthMode, onEnter }) {
  const isLogin = authMode === 'login'
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const submit = async (event) => {
    event.preventDefault()
    setError('')
    setSubmitting(true)
    const values = Object.fromEntries(new FormData(event.currentTarget))
    try {
      const user = isLogin
        ? await api.login(values.email, values.password)
        : await api.register(values.name, values.email, values.phone, values.password)
      onEnter(user)
    } catch (requestError) {
      setError(requestError.message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="auth-screen">
      <section className="auth-panel">
        <div className="auth-copy">
          <div className="brand large">
            <div className="brand-mark">
              <Wifi size={20} />
            </div>
            <div>
              <strong>eSIM Control</strong>
              <span>Операционная CRM</span>
            </div>
          </div>
          <h1>Контроль заказов, eSIM, поставщиков и прибыли в одном интерфейсе</h1>
          <div className="auth-stats">
            <Metric icon={ShoppingCart} label="Заказов сегодня" value="128" />
            <Metric icon={Smartphone} label="Активных eSIM" value="4 812" />
            <Metric icon={CircleDollarSign} label="Чистая прибыль" value="1.42M" />
          </div>
        </div>

        <form className="auth-card" onSubmit={submit}>
          <div className="segmented">
            <button type="button" className={isLogin ? 'selected' : ''} onClick={() => setAuthMode('login')}>
              Вход
            </button>
            <button type="button" className={!isLogin ? 'selected' : ''} onClick={() => setAuthMode('register')}>
              Регистрация
            </button>
          </div>
          <div>
            <h2>{isLogin ? 'Войти в CRM' : 'Создать аккаунт'}</h2>
            <p>{isLogin ? 'Доступ для сотрудников и администраторов' : 'Первичный аккаунт владельца CRM'}</p>
          </div>
          {!isLogin && <Field icon={UserPlus} label="Имя" name="name" placeholder="Джон" />}
          <Field icon={Mail} label="Email" name="email" placeholder="admin@ecrm.local" />
          {!isLogin && <Field icon={Phone} label="Телефон" name="phone" placeholder="+7 700 000 00 00" />}
          <Field icon={Lock} label="Пароль" name="password" placeholder="Введите пароль" type="password" />
          {error && <div className="notice danger"><strong>{error}</strong></div>}
          <button className="primary full" type="submit" disabled={submitting}>
            <KeyRound size={18} />
            {submitting ? '...' : isLogin ? 'Войти' : 'Зарегистрироваться'}
          </button>
        </form>
      </section>
    </main>
  )
}

function Field({ icon: Icon, label, name, placeholder, type = 'text' }) {
  return (
    <label className="field">
      <span>{label}</span>
      <div>
        <Icon size={17} />
        <input type={type} name={name} placeholder={placeholder} required />
      </div>
    </label>
  )
}

function Topbar({ activeItem, query, setQuery, matches, onMenu, onNavigate, onLogout, user }) {
  const Icon = activeItem.icon

  return (
    <header className="topbar">
      <div className="page-title">
        <button className="icon-button mobile-only" type="button" onClick={onMenu} title="Меню">
          <Menu size={20} />
        </button>
        <div className="title-icon">
          <Icon size={19} />
        </div>
        <div>
          <h1>{activeItem.label}</h1>
          <span>Алматы Office / Primary</span>
        </div>
      </div>

      <div className="search-box">
        <Search size={18} />
        <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Поиск: клиент, ICID, заказ, тариф" />
        {query && (
          <button type="button" onClick={() => setQuery('')} title="Очистить">
            <X size={16} />
          </button>
        )}
        {matches.length > 0 && (
          <div className="search-results">
            {matches.map((item) => (
              <button type="button" key={`${item.type}-${item.title}`} onClick={() => { onNavigate(item.view); setQuery('') }}>
                <span>{item.type}</span>
                <strong>{item.title}</strong>
                <em>{item.meta}</em>
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="top-actions">
        <button className="balance-button" type="button">
          <CreditCard size={17} />
          <span>Баланс</span>
          <strong>{money.format(4546300)}</strong>
          <ChevronDown size={16} />
        </button>
        <button className="icon-button" type="button" title="Уведомления">
          <Bell size={19} />
        </button>
        <button className="user-menu" type="button" onClick={onLogout}>
          <span>{user?.name?.slice(0, 1)?.toUpperCase() || 'U'}</span>
          <strong>{user?.name || 'User'}</strong>
          <LogOut size={16} />
        </button>
      </div>
    </header>
  )
}

function ViewRouter({ activeView, setActiveView, setSelectedOrder }) {
  if (activeView === 'orders') return <OrdersView setActiveView={setActiveView} setSelectedOrder={setSelectedOrder} />
  if (activeView === 'new-order') return <NewOrderView />
  if (activeView === 'customers') return <CustomersView />
  if (activeView === 'esims') return <EsimsView setSelectedOrder={setSelectedOrder} />
  if (activeView === 'tariffs') return <TariffsView />
  if (activeView === 'suppliers') return <SuppliersView />
  if (activeView === 'sales') return <SalesView />
  if (activeView === 'finance') return <FinanceView />
  if (activeView === 'reports') return <ReportsView />
  if (activeView === 'api') return <ApiView />
  if (activeView === 'notifications') return <NotificationsView />
  if (activeView === 'employees') return <EmployeesView />
  if (activeView === 'activity') return <ActivityView />
  return <Dashboard setActiveView={setActiveView} setSelectedOrder={setSelectedOrder} />
}

function Dashboard({ setActiveView, setSelectedOrder }) {
  const { salesTrend, countries, supplierBars, notifications, orders } = useCRM()
  return (
    <div className="page-grid">
      <section className="kpi-grid">
        <Kpi icon={Users} label="Все клиенты" value={number.format(6284)} delta="+18 сегодня" tone="teal" />
        <Kpi icon={Smartphone} label="Активные eSIM" value={number.format(4812)} delta="78% от проданных" tone="blue" />
        <Kpi icon={ShoppingCart} label="Активные заказы" value="312" delta="+42 за неделю" tone="orange" />
        <Kpi icon={CircleDollarSign} label="Чистая прибыль" value={money.format(1424000)} delta="+12.4% месяц" tone="green" />
        <Kpi icon={PackageCheck} label="Свободные eSIM" value={number.format(864)} delta="у 4 поставщиков" tone="violet" />
        <Kpi icon={Banknote} label="Средний чек" value={money.format(8220)} delta="за 7 дней" tone="red" />
      </section>

      <section className="panel wide">
        <PanelHeader
          icon={BarChart3}
          title="Продажи и прибыль"
          action={<PeriodTabs />}
        />
        <div className="chart-area">
          <ResponsiveContainer width="100%" height={310}>
            <AreaChart data={salesTrend} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="revenueFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#0f766e" stopOpacity={0.26} />
                  <stop offset="95%" stopColor="#0f766e" stopOpacity={0.02} />
                </linearGradient>
                <linearGradient id="profitFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#e95d2a" stopOpacity={0.22} />
                  <stop offset="95%" stopColor="#e95d2a" stopOpacity={0.02} />
                </linearGradient>
              </defs>
              <CartesianGrid stroke="#e7ece9" vertical={false} />
              <XAxis dataKey="date" axisLine={false} tickLine={false} />
              <YAxis axisLine={false} tickLine={false} tickFormatter={(value) => `${value / 1000}k`} />
              <Tooltip formatter={(value) => money.format(value)} />
              <Area type="monotone" dataKey="revenue" name="Выручка" stroke="#0f766e" fill="url(#revenueFill)" strokeWidth={3} />
              <Area type="monotone" dataKey="profit" name="Прибыль" stroke="#e95d2a" fill="url(#profitFill)" strokeWidth={3} />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </section>

      <section className="panel">
        <PanelHeader icon={Globe2} title="Топ стран" action={<button className="ghost" type="button">Все</button>} />
        <div className="pie-wrap">
          <ResponsiveContainer width="100%" height={210}>
            <PieChart>
              <Pie data={countries} innerRadius={58} outerRadius={88} dataKey="value" paddingAngle={3}>
                {countries.map((entry) => <Cell key={entry.name} fill={entry.fill} />)}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
          <div className="legend-list">
            {countries.map((item) => (
              <div key={item.name}>
                <i style={{ background: item.fill }} />
                <span>{item.name}</span>
                <strong>{item.value}%</strong>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="panel">
        <PanelHeader icon={Truck} title="Поставщики" action={<button className="ghost" type="button" onClick={() => setActiveView('suppliers')}>Открыть</button>} />
        <ResponsiveContainer width="100%" height={270}>
          <BarChart data={supplierBars} margin={{ top: 10, right: 8, left: -20, bottom: 0 }}>
            <CartesianGrid stroke="#e7ece9" vertical={false} />
            <XAxis dataKey="name" axisLine={false} tickLine={false} />
            <YAxis axisLine={false} tickLine={false} />
            <Tooltip />
            <Bar dataKey="stock" name="Остаток" fill="#b9c7c1" radius={[6, 6, 0, 0]} />
            <Bar dataKey="sold" name="Продано" fill="#0f766e" radius={[6, 6, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </section>

      <section className="panel wide">
        <PanelHeader
          icon={AlertTriangle}
          title="Контроль операций"
          action={<button className="ghost" type="button" onClick={() => setActiveView('activity')}>История</button>}
        />
        <div className="ops-grid">
          {notifications.map((item) => (
            <div className={`notice ${item.level}`} key={item.text}>
              <span>{item.type}</span>
              <strong>{item.text}</strong>
            </div>
          ))}
        </div>
        <OrdersTable compact setSelectedOrder={setSelectedOrder} rows={orders.slice(0, 3)} />
      </section>
    </div>
  )
}

function OrdersView({ setActiveView, setSelectedOrder }) {
  const { orders } = useCRM()
  return (
    <section className="panel full">
      <div className="toolbar">
        <div className="button-group">
          <button className="primary" type="button">
            <Check size={17} />
            Подтвердить заявки
          </button>
          <button className="secondary" type="button">
            <X size={17} />
            Отменить заявки
          </button>
        </div>
        <div className="toolbar-spacer" />
        <button className="secondary" type="button">
          <Filter size={17} />
          Фильтр
        </button>
        <button className="primary" type="button" onClick={() => setActiveView('new-order')}>
          <Plus size={17} />
          Создать новую
        </button>
      </div>
      <OrdersTable setSelectedOrder={setSelectedOrder} rows={orders} />
    </section>
  )
}

function OrdersTable({ rows, compact = false, setSelectedOrder }) {
  return (
    <div className="table-wrap">
      <table className={compact ? 'data-table compact' : 'data-table'}>
        <thead>
          <tr>
            <th><input type="checkbox" aria-label="Выбрать все" /></th>
            <th>ID</th>
            <th>Клиент</th>
            <th>Дата</th>
            <th>Поставщик</th>
            <th>Тариф</th>
            <th>ICCID</th>
            <th>Оплата</th>
            <th>Статус</th>
            <th>Прибыль</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {rows.map((order) => (
            <tr key={order.id}>
              <td><input type="checkbox" aria-label={`Выбрать ${order.id}`} /></td>
              <td><strong>{order.id}</strong></td>
              <td>
                <button className="link-button" type="button">{order.customer}</button>
                <small>{order.phone}</small>
              </td>
              <td>{order.created}</td>
              <td>{order.provider}</td>
              <td>{order.country} {order.plan}</td>
              <td><span className="iccid">{order.iccid}</span></td>
              <td><Status text={order.paymentStatus} /></td>
              <td><Status text={order.status} /></td>
              <td>{formatMoney(order.profit, order.currency)}</td>
              <td>
                <button className="icon-button" type="button" onClick={() => setSelectedOrder(order)} title="QR и детали">
                  <QrCode size={18} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function NewOrderView() {
  const { tariffs, refreshData } = useCRM()
  const [selectedPlan, setSelectedPlan] = useState(tariffs[0])
  const [phone, setPhone] = useState('')
  const [name, setName] = useState('')
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (!tariffs.some((item) => item.id === selectedPlan?.id)) setSelectedPlan(tariffs[0])
  }, [tariffs, selectedPlan])

  const createOrder = async () => {
    if (!selectedPlan?._id || !phone.trim() || !name.trim()) {
      setMessage('Заполните телефон, ФИО и выберите тариф')
      return
    }
    try {
      const order = await api.createOrder({ customer: { name, phone, country: 'Казахстан' }, tariffId: selectedPlan._id, paymentMethod: 'kaspi_qr', paymentStatus: 'paid' })
      setMessage(`Заказ #${order.number} создан, ICCID ${order.iccid}`)
      setPhone('')
      setName('')
      await refreshData()
    } catch (error) {
      setMessage(error.message)
    }
  }

  return (
    <section className="panel full order-builder">
      <PanelHeader icon={Plus} title="Создать заказ на eSIM" action={<button className="secondary" type="button"><Download size={17} /> Импорт ICID</button>} />
      <div className="form-grid">
        <label className="field wide">
          <span>Регион группы</span>
          <div>
            <Globe2 size={17} />
            <select defaultValue="asia">
              <option value="asia">Азия 4</option>
              <option value="europe">Европа 33</option>
              <option value="gcc">ОАЭ и GCC</option>
            </select>
          </div>
        </label>
        <label className="field">
          <span>Телефон</span>
          <div>
            <Phone size={17} />
            <input placeholder="+7" value={phone} onChange={(event) => setPhone(event.target.value)} />
          </div>
        </label>
        <label className="field">
          <span>ФИО</span>
          <div>
            <Users size={17} />
            <input placeholder="Введите ФИО" value={name} onChange={(event) => setName(event.target.value)} />
          </div>
        </label>
      </div>

      <div className="tariff-picker">
        {tariffs.map((tariff) => (
          <button
            key={tariff.id}
            className={selectedPlan.id === tariff.id ? 'tariff-card selected' : 'tariff-card'}
            type="button"
            onClick={() => setSelectedPlan(tariff)}
          >
            <span>{tariff.country}</span>
            <strong>{tariff.gb} · {tariff.days}</strong>
            <b>{formatMoney(tariff.price, tariff.currency)}</b>
            <small>Себестоимость {formatMoney(tariff.cost, tariff.currency)}</small>
          </button>
        ))}
      </div>

      <section className="order-summary">
        <div>
          <span>Выбранный тариф</span>
          <strong>{selectedPlan.country} / {selectedPlan.gb} / {selectedPlan.days}</strong>
        </div>
        <div>
          <span>Поставщик</span>
          <strong>{selectedPlan.provider}</strong>
        </div>
        <div>
          <span>Расчет прибыли</span>
          <strong>{formatMoney(selectedPlan.price - selectedPlan.cost, selectedPlan.currency)}</strong>
        </div>
        <button className="primary" type="button" onClick={createOrder}>
          <PackageCheck size={18} />
          Создать заказ
        </button>
      </section>
      {message && <div className="notice info"><strong>{message}</strong></div>}
    </section>
  )
}

function CustomersView() {
  const { customers } = useCRM()
  return (
    <section className="panel full">
      <PanelHeader icon={Users} title="Клиенты" action={<button className="primary" type="button"><UserPlus size={17} /> Новый клиент</button>} />
      <div className="cards-row">
        <Metric icon={Users} label="Новые за день" value="18" />
        <Metric icon={Calendar} label="Новые за месяц" value="642" />
        <Metric icon={Banknote} label="LTV средний" value={money.format(18400)} />
      </div>
      <GenericTable
        columns={['ID', 'ФИО', 'Телефон', 'Email', 'Страна', 'Заказы', 'Сумма', 'Прибыль', 'Статус']}
        rows={customers.map((item) => [item.id, item.name, item.phone, item.email, item.country, item.orders, money.format(item.total), money.format(item.profit), <Status text={item.status} />])}
      />
    </section>
  )
}

function EsimsView({ setSelectedOrder }) {
  const { esims, orders } = useCRM()
  return (
    <section className="panel full">
      <PanelHeader icon={Smartphone} title="База eSIM" action={<button className="secondary" type="button"><RefreshCcw size={17} /> Синхронизация</button>} />
      <div className="cards-row">
        <Metric icon={BadgeCheck} label="Active" value="4 812" />
        <Metric icon={Database} label="Available" value="864" />
        <Metric icon={PackageCheck} label="Sold" value="6 928" />
        <Metric icon={AlertTriangle} label="Expired" value="214" />
      </div>
      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>ICCID</th>
              <th>IMSI</th>
              <th>Страна</th>
              <th>Тариф</th>
              <th>Остаток</th>
              <th>Статус</th>
              <th>Поставщик</th>
              <th>Клиент</th>
              <th>Прибыль</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {esims.map((item) => (
              <tr key={item.id}>
                <td><strong>{item.id}</strong></td>
                <td><span className="iccid">{item.iccid}</span></td>
                <td>{item.imsi}</td>
                <td>{item.country}</td>
                <td>{item.plan}</td>
                <td>{item.traffic}</td>
                <td><Status text={item.status} /></td>
                <td>{item.provider}</td>
                <td>{item.customer}</td>
                <td>{money.format(item.profit)}</td>
                <td>
                  {item.order !== '-' && (
                    <button className="icon-button" type="button" onClick={() => setSelectedOrder(orders.find((order) => order.id === item.order))} title="QR">
                      <QrCode size={18} />
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}

function TariffsView() {
  const { tariffs } = useCRM()
  return (
    <section className="panel full">
      <PanelHeader icon={Tags} title="Тарифы" action={<button className="primary" type="button"><Plus size={17} /> Добавить тариф</button>} />
      <div className="tariff-grid">
        {tariffs.map((tariff) => (
          <article className="tariff-tile" key={tariff.id}>
            <div>
              <span>{tariff.id}</span>
              <Status text={tariff.status} />
            </div>
            <h3>{tariff.name}</h3>
            <strong>{tariff.country} · {tariff.gb} · {tariff.days}</strong>
            <dl>
              <dt>Поставщик</dt><dd>{tariff.provider}</dd>
              <dt>Себестоимость</dt><dd>{formatMoney(tariff.cost, tariff.currency)}</dd>
              <dt>Продажа</dt><dd>{formatMoney(tariff.price, tariff.currency)}</dd>
              <dt>Прибыль</dt><dd>{formatMoney(tariff.price - tariff.cost, tariff.currency)}</dd>
              <dt>Остаток</dt><dd>{tariff.stock}</dd>
            </dl>
          </article>
        ))}
      </div>
    </section>
  )
}

function SuppliersView() {
  const { suppliers } = useCRM()
  return (
    <section className="panel full">
      <PanelHeader icon={Truck} title="Поставщики" action={<button className="primary" type="button"><Plus size={17} /> Новый поставщик</button>} />
      <div className="supplier-grid">
        {suppliers.map((supplier) => (
          <article className="supplier-card" key={supplier.id}>
            <header>
              <div className="supplier-logo">{supplier.name.slice(0, 2)}</div>
              <div>
                <h3>{supplier.name}</h3>
                <span>{supplier.contact}</span>
              </div>
              <Status text={supplier.status} />
            </header>
            <div className="supplier-stats">
              <Metric icon={CreditCard} label="Баланс" value={formatMoney(supplier.balance, supplier.currency)} />
              <Metric icon={Globe2} label="Страны" value={supplier.countries} />
              <Metric icon={Smartphone} label="eSIM" value={supplier.esims} />
            </div>
            <div className="integration-row">
              <span>API Status</span>
              <Status text={supplier.api} />
              <strong>{supplier.terms}</strong>
            </div>
          </article>
        ))}
      </div>
    </section>
  )
}

function SalesView() {
  const { salesTrend } = useCRM()
  return (
    <section className="panel full">
      <PanelHeader icon={BarChart3} title="Продажи" action={<PeriodTabs />} />
      <div className="cards-row">
        <Metric icon={ShoppingCart} label="Кол-во продаж" value="436" />
        <Metric icon={Banknote} label="Выручка" value={money.format(1804000)} />
        <Metric icon={Database} label="Себестоимость" value={money.format(1011000)} />
        <Metric icon={CircleDollarSign} label="Чистая прибыль" value={money.format(641000)} />
      </div>
      <div className="chart-area large">
        <ResponsiveContainer width="100%" height={360}>
          <AreaChart data={salesTrend}>
            <CartesianGrid stroke="#e7ece9" vertical={false} />
            <XAxis dataKey="date" axisLine={false} tickLine={false} />
            <YAxis axisLine={false} tickLine={false} tickFormatter={(value) => `${value / 1000}k`} />
            <Tooltip formatter={(value) => money.format(value)} />
            <Area type="monotone" dataKey="revenue" name="Выручка" stroke="#2563eb" fill="#dbeafe" strokeWidth={3} />
            <Area type="monotone" dataKey="cost" name="Себестоимость" stroke="#64748b" fill="#e2e8f0" strokeWidth={2} />
            <Area type="monotone" dataKey="profit" name="Прибыль" stroke="#16a34a" fill="#dcfce7" strokeWidth={3} />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </section>
  )
}

function FinanceView() {
  const { orders } = useCRM()
  const rows = orders.map((order) => [
    order.id,
    order.customer,
    formatMoney(order.price, order.currency),
    formatMoney(order.cost, order.currency),
    formatMoney(order.fee, order.currency),
    formatMoney(order.tax, order.currency),
    formatMoney(order.profit, order.currency),
  ])

  return (
    <section className="panel full">
      <PanelHeader icon={CircleDollarSign} title="Финансы" action={<button className="secondary" type="button"><Download size={17} /> Экспорт</button>} />
      <div className="finance-formula">
        <span>Цена продажи</span>
        <b>-</b>
        <span>Себестоимость</span>
        <b>-</b>
        <span>Комиссия</span>
        <b>-</b>
        <span>Налог</span>
        <b>=</b>
        <strong>Чистая прибыль</strong>
      </div>
      <div className="cards-row">
        <Metric icon={Banknote} label="Выручка" value={money.format(1804000)} />
        <Metric icon={Database} label="Расходы" value={money.format(1011000)} />
        <Metric icon={CreditCard} label="Комиссии" value={money.format(54120)} />
        <Metric icon={CircleDollarSign} label="Валовая прибыль" value={money.format(793000)} />
      </div>
      <GenericTable columns={['Order ID', 'Клиент', 'Продажа', 'Себестоимость', 'Комиссия', 'Налог', 'Чистая прибыль']} rows={rows} />
    </section>
  )
}

function ReportsView() {
  const reportRows = [
    ['R-01', 'Продажи по дням', 'Продажи', 'Сегодня 09:00', 'Готов'],
    ['R-02', 'Прибыль по поставщикам', 'Поставщики', 'Сегодня 08:30', 'Готов'],
    ['R-03', 'LTV и повторные покупки', 'Клиенты', 'Вчера 19:20', 'Готов'],
    ['R-04', 'Маржинальность тарифов', 'Финансы', 'Каждый понедельник', 'Запланирован'],
  ]

  return (
    <section className="panel full">
      <PanelHeader icon={FileText} title="Отчеты" action={<button className="primary" type="button"><Download size={17} /> Скачать отчет</button>} />
      <div className="cards-row">
        <Metric icon={BarChart3} label="Продажи по странам" value="5 групп" />
        <Metric icon={Truck} label="Поставщики" value="3 отчета" />
        <Metric icon={Users} label="Клиенты" value="LTV / Retention" />
        <Metric icon={CircleDollarSign} label="Прибыль" value="День / месяц" />
      </div>
      <GenericTable columns={['ID', 'Название', 'Раздел', 'Обновлен', 'Статус']} rows={reportRows.map((row) => [...row.slice(0, 4), <Status text={row[4]} />])} />
    </section>
  )
}

function ApiView() {
  const { suppliers, refreshData } = useCRM()
  const [syncing, setSyncing] = useState(false)
  const [syncMessage, setSyncMessage] = useState('')
  const sync = async () => {
    setSyncing(true)
    setSyncMessage('')
    try {
      const result = await api.syncZesimo()
      setSyncMessage(`Zesimo: синхронизировано ${result.packages} тарифов, баланс ${result.balance} ${result.currency}`)
      await refreshData()
    } catch (error) {
      setSyncMessage(error.message)
    } finally {
      setSyncing(false)
    }
  }
  const apiRows = suppliers.map((supplier) => [
    supplier.id,
    supplier.name,
    <Status text={supplier.api} />,
    `${supplier.countries} стран`,
    `${supplier.esims} eSIM`,
    formatMoney(supplier.balance, supplier.currency),
    supplier.api === 'Warning' ? 'Требуется проверка баланса' : 'Синхронизирован 2 мин назад',
  ])

  return (
    <section className="panel full">
      <PanelHeader icon={Database} title="API поставщиков" action={<button className="secondary" type="button" onClick={sync} disabled={syncing}><RefreshCcw size={17} /> {syncing ? 'Синхронизация…' : 'Обновить Zesimo'}</button>} />
      {syncMessage && <div className="notice info"><strong>{syncMessage}</strong></div>}
      <div className="api-flow">
        {['Оплата', 'Выбор поставщика', 'Покупка eSIM', 'ICCID / QR', 'Отправка клиенту'].map((step, index) => (
          <div key={step}>
            <span>{index + 1}</span>
            <strong>{step}</strong>
          </div>
        ))}
      </div>
      <GenericTable columns={['ID', 'Поставщик', 'API', 'Покрытие', 'Остаток', 'Баланс', 'Последний статус']} rows={apiRows} />
    </section>
  )
}

function NotificationsView() {
  const { notifications } = useCRM()
  const rows = notifications.map((item, index) => [
    `N-${String(index + 1).padStart(3, '0')}`,
    item.type,
    item.text,
    <Status text={item.level === 'danger' ? 'Критично' : item.level === 'warning' ? 'Внимание' : 'Инфо'} />,
    '27.08.2026 15:00',
  ])

  return (
    <section className="panel full">
      <PanelHeader icon={Bell} title="Уведомления" action={<button className="secondary" type="button"><Check size={17} /> Отметить прочитанным</button>} />
      <div className="cards-row">
        <Metric icon={AlertTriangle} label="Критичные" value="1" />
        <Metric icon={PackageCheck} label="Склад" value="1" />
        <Metric icon={Database} label="API события" value="1" />
        <Metric icon={Bell} label="Всего сегодня" value="9" />
      </div>
      <GenericTable columns={['ID', 'Тип', 'Сообщение', 'Приоритет', 'Время']} rows={rows} />
    </section>
  )
}

function EmployeesView() {
  const { employees } = useCRM()
  return (
    <section className="panel full">
      <PanelHeader icon={ShieldCheck} title="Сотрудники и права" action={<button className="primary" type="button"><UserPlus size={17} /> Добавить</button>} />
      <GenericTable
        columns={['ID', 'Имя', 'Телефон', 'Email', 'Роль', 'Логин', 'Статус', 'Доступ']}
        rows={employees.map((item) => [item.id, item.name, item.phone, item.email, item.role, item.login, <Status text={item.status} />, item.access])}
      />
    </section>
  )
}

function ActivityView() {
  const { activities } = useCRM()
  return (
    <section className="panel full">
      <PanelHeader icon={Activity} title="История действий" action={<button className="secondary" type="button"><Filter size={17} /> Фильтр</button>} />
      <div className="timeline">
        {activities.map((item) => (
          <article key={`${item.action}-${item.time}`}>
            <div className="timeline-dot" />
            <div>
              <span>{item.time}</span>
              <h3>{item.action}</h3>
              <p>{item.user}: {item.oldValue} → {item.newValue}</p>
            </div>
          </article>
        ))}
      </div>
    </section>
  )
}

function GenericTable({ columns, rows }) {
  return (
    <div className="table-wrap">
      <table className="data-table">
        <thead>
          <tr>
            {columns.map((column) => <th key={column}>{column}</th>)}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr key={`${row[0]}-${index}`}>
              {row.map((cell, cellIndex) => <td key={`${row[0]}-${cellIndex}`}>{cell}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function Kpi({ icon: Icon, label, value, delta, tone }) {
  return (
    <article className={`kpi ${tone ?? ''}`}>
      <div className="kpi-icon">
        <Icon size={20} />
      </div>
      <div>
        <span>{label}</span>
        <strong>{value}</strong>
        {delta && <em>{delta}</em>}
      </div>
    </article>
  )
}

function Metric({ icon: Icon, label, value }) {
  return (
    <div className="metric">
      <Icon size={18} />
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function PanelHeader({ icon: Icon, title, action }) {
  return (
    <header className="panel-header">
      <div>
        <Icon size={19} />
        <h2>{title}</h2>
      </div>
      {action}
    </header>
  )
}

function PeriodTabs() {
  return (
    <div className="period-tabs">
      {['День', 'Неделя', 'Месяц'].map((item, index) => (
        <button className={index === 1 ? 'selected' : ''} key={item} type="button">{item}</button>
      ))}
    </div>
  )
}

function Status({ text }) {
  const normalized = `${text}`.toLowerCase()
  let tone = 'neutral'
  if (['active', 'activated', 'completed', 'оплачен', 'активный', 'vip', 'online', 'работает', 'активен'].some((value) => normalized.includes(value))) tone = 'success'
  if (['pending', 'ожидает', 'reserved', 'мало', 'warning', 'away', 'низкий'].some((value) => normalized.includes(value))) tone = 'warning'
  if (['cancelled', 'refunded', 'expired', 'danger'].some((value) => normalized.includes(value))) tone = 'danger'
  if (['assigned', 'available'].some((value) => normalized.includes(value))) tone = 'info'

  return <span className={`status ${tone}`}>{text}</span>
}

function ActivationModal({ order, onClose }) {
  const qrBits = Array.from({ length: 121 }, (_, index) => (index * 17 + 9) % 7 < 3 || [0, 1, 2, 11, 22, 98, 108, 118, 120].includes(index))

  return (
    <div className="modal-backdrop" role="presentation">
      <section className="activation-modal" role="dialog" aria-modal="true" aria-labelledby="activation-title">
        <header>
          <div>
            <div className="title-icon light">
              <QrCode size={19} />
            </div>
            <div>
              <h2 id="activation-title">eSIM активация</h2>
              <span>{order.id} · {order.customer}</span>
            </div>
          </div>
          <button className="icon-button light" type="button" onClick={onClose} title="Закрыть">
            <X size={19} />
          </button>
        </header>

        <div className="modal-body">
          <div className="qr-card">
            <div className="qr-grid" aria-label="QR код">
              {qrBits.map((filled, index) => <i className={filled ? 'filled' : ''} key={index} />)}
            </div>
            <span>Сканируйте камерой телефона</span>
          </div>

          <section className="install-box">
            <div>
              <Send size={18} />
              <div>
                <strong>Установка в один клик</strong>
                <span>Откройте ссылку на телефоне или отправьте клиенту</span>
              </div>
            </div>
            <button className="dark-button" type="button" disabled={!order.iosTapLink && !order.installUrl} onClick={() => window.open(order.iosTapLink || order.installUrl, '_blank', 'noopener,noreferrer')}><Apple size={17} /> Установить на iPhone</button>
            <button className="android-button" type="button" disabled={!order.androidTapLink && !order.installUrl} onClick={() => window.open(order.androidTapLink || order.installUrl, '_blank', 'noopener,noreferrer')}><Smartphone size={17} /> Установить на Android</button>
          </section>

          <InfoLine label="ICCID" value={order.iccid} />
          <InfoLine label="SM-DP+ ADDRESS" value="rsp.esim-control.net" />
          <InfoLine label="Activation Code" value={`LPA:1$${order.iccid}$${order.country.toUpperCase()}`} />
          <InfoLine label="Тариф" value={`${order.country} · ${order.plan}`} />
        </div>

        <footer>
          <button className="secondary" type="button" onClick={onClose}>
            Закрыть
          </button>
        </footer>
      </section>
    </div>
  )
}

function InfoLine({ label, value }) {
  return (
    <div className="info-line">
      <span>{label}</span>
      <strong>{value}</strong>
      <button className="icon-button" type="button" title="Скопировать">
        <Copy size={16} />
      </button>
    </div>
  )
}

export default App
