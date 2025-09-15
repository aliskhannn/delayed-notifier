import NotificationForm from "./components/NotificationForm";
import NotificationList from "./components/NotificationList";

export default function App() {
  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-5xl mx-auto">
        <header className="mb-6">
          <h1 className="text-3xl font-semibold text-gray-800">
            DelayedNotifier — UI
          </h1>
          <p className="text-sm text-gray-500">
            Создавайте отложенные уведомления, смотрите статусы и отменяйте.
          </p>
        </header>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="bg-white p-6 rounded-2xl shadow">
            <NotificationForm />
          </div>

          <div className="bg-white p-6 rounded-2xl shadow">
            <NotificationList />
          </div>
        </div>
      </div>
    </div>
  );
}
