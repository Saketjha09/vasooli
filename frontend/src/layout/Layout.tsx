import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'

/** Wraps every route — the sidebar is genuinely persistent, not per-page chrome. */
export function Layout() {
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <main className="min-w-0 flex-1">
        <Outlet />
      </main>
    </div>
  )
}
