import { useEffect, useState } from "react";
import type { Notification } from "../entities/notification";

const statusClass = (s: string) => {
  switch (s) {
    case "pending":
      return "bg-amber-100 text-amber-800";
    case "sent":
      return "bg-green-100 text-green-800";
    case "cancelled":
      return "bg-gray-100 text-gray-800";
    case "failed":
      return "bg-red-100 text-red-800";
    default:
      return "bg-gray-100 text-gray-800";
  }
};

export default function NotificationList() {
  const [items, setItems] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchList = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`http://localhost:8080/api/notify/`, {
        method: "GET",
      });
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const data = await res.json();
      setItems(Array.isArray(data.result) ? data.result : []);
    } catch (err: any) {
      setError(err.message ?? "Ошибка");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchList();
    const id = setInterval(fetchList, 10000); // опрашиваем каждые 10s для обновления статусов
    return () => clearInterval(id);
  }, []);

  const handleCancel = async (id: string) => {
    if (!confirm("Отменить уведомление?")) return;
    try {
      const res = await fetch(`http://localhost:8080/api/notify/${id}`, {
        method: "DELETE",
      });
      if (!res.ok) throw new Error("Не удалось отменить");
      await fetchList();
    } catch (err: any) {
      alert("Ошибка: " + (err.message ?? err));
    }
  };

  return (
    <>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-medium">Уведомления</h2>
        <div>
          <button
            onClick={fetchList}
            className="text-sm px-3 py-1 border rounded cursor-pointer"
          >
            Обновить
          </button>
        </div>
      </div>

      {loading && <div className="text-sm text-gray-500">Загрузка...</div>}
      {error && <div className="text-sm text-red-600">Ошибка: {error}</div>}

      <div className="space-y-3">
        {items.length === 0 && !loading ? (
          <div className="text-sm text-gray-500">Нет уведомлений</div>
        ) : (
          items.map((item) => (
            <div
              key={item.id}
              className="p-3 border rounded flex justify-between items-start"
            >
              <div style={{ minWidth: 0 }} className="pr-4">
                <div className="flex items-center gap-2">
                  <div
                    className={`px-2 py-1 text-xs font-semibold rounded ${statusClass(
                      item.status
                    )}`}
                  >
                    {item.status}
                  </div>
                  <div className="text-sm text-gray-600 truncate">
                    {item.channel} → {item.to}
                  </div>
                </div>
                <div className="mt-2 text-sm text-gray-800">{item.message}</div>
                <div className="mt-1 text-xs text-gray-500">
                  send_at: {item.send_at}
                </div>
              </div>

              <div className="flex flex-col items-end gap-2">
                <div className="text-xs text-gray-500">
                  {item.retries} retries
                </div>
                {item.status === "pending" && (
                  <button
                    onClick={() => handleCancel(item.id)}
                    className="text-sm px-3 py-1 bg-red-600 text-white rounded"
                  >
                    Cancel
                  </button>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </>
  );
}
