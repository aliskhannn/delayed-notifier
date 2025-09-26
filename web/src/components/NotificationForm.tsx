import React, { useState } from 'react'
import type { Channel } from '../entities/notification'

export default function NotificationForm() {
  const [message, setMessage] = useState('Hello! This is a test notification 1.')
  const [sendAt, setSendAt] = useState<string>(() => {
    const now = new Date()
    now.setMinutes(now.getMinutes() + 10)
    // yyyy-mm-ddThh:mm for input type=datetime-local
    return now.toISOString().slice(0,16)
  })
  const [retries, setRetries] = useState<number>(3)
  const [to, setTo] = useState('')
  const [channel, setChannel] = useState<Channel>('telegram')
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setMsg(null)
    try {
      // convert datetime-local to "YYYY-MM-DD HH:mm:ss"
      const dt = new Date(sendAt)
      const pad = (n: number) => n.toString().padStart(2, '0')
      const formatted = `${dt.getFullYear()}-${pad(dt.getMonth()+1)}-${pad(dt.getDate())} ${pad(dt.getHours())}:${pad(dt.getMinutes())}:${pad(dt.getSeconds())}`

      const body = {
        message,
        send_at: formatted,
        retries,
        to,
        channel
      }

      const res = await fetch(`http://localhost:8080/api/notify/`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      })
      if (!res.ok) {
        const txt = await res.text()
        throw new Error(txt || res.statusText)
      }
      setMsg('Уведомление создано')
      // reset small fields
      setTo('')
    } catch (err: any) {
      setMsg('Ошибка: ' + (err.message ?? err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <h2 className="text-xl font-medium mb-4">Создать уведомление</h2>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700">Message</label>
          <textarea value={message} onChange={e => setMessage(e.target.value)} rows={3}
            className="mt-1 block w-full border rounded p-2" />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">Send at</label>
          <input
            type="datetime-local"
            value={sendAt}
            onChange={e => setSendAt(e.target.value)}
            className="mt-1 block w-full border rounded p-2"
            required
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-sm font-medium text-gray-700">Retries</label>
            <input type="number" min={0} value={retries} onChange={e => setRetries(Number(e.target.value))}
              className="mt-1 block w-full border rounded p-2" />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700">Channel</label>
            <select value={channel} onChange={e => setChannel(e.target.value as Channel)}
              className="mt-1 block w-full border rounded p-2">
              <option value="telegram">Telegram</option>
              <option value="email">Email</option>
            </select>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">{channel === 'telegram' ? 'Telegram chat id' : 'Email'}</label>
          <input value={to} onChange={e => setTo(e.target.value)} required
            className="mt-1 block w-full border rounded p-2" placeholder={channel === 'telegram' ? '7888928504' : 'user@example.com'} />
        </div>

        <div className="flex items-center gap-3">
          <button type="submit" disabled={loading}
            className="px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700 disabled:opacity-60 cursor-pointer">
            {loading ? 'Sending...' : 'Create'}
          </button>
          {msg && <div className="text-sm text-gray-600">{msg}</div>}
        </div>
      </form>
    </>
  )
}
