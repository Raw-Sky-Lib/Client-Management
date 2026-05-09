import { createBrowserRouter, Navigate } from 'react-router'
import { GuestRoute } from '@/components/guards/GuestRoute'
import { ProtectedRoute } from '@/components/guards/ProtectedRoute'
import { ConnectPage } from '@/features/onboarding/connect-page'

// Placeholders — replaced as each milestone builds the real component
const Placeholder = ({ name }: { name: string }) => (
  <div className="flex min-h-svh items-center justify-center bg-gray-50">
    <p className="text-sm text-gray-400">{name}</p>
  </div>
)

export const router = createBrowserRouter([
  {
    index: true,
    element: <Navigate to="/dashboard" replace />,
  },

  // ─── Guest routes ─────────────────────────────────────────────────────────
  {
    element: <GuestRoute />,
    children: [
      { path: '/connect', element: <ConnectPage /> },
    ],
  },

  // Auth callback — no guard; handles token exchange then redirects
  {
    path: '/auth/callback',
    element: <Placeholder name="AuthCallbackPage" />,
  },

  // ─── Protected routes ─────────────────────────────────────────────────────
  {
    element: <ProtectedRoute />,
    children: [
      { path: '/dashboard',       element: <Placeholder name="DashboardPage" /> },
      { path: '/pages',           element: <Placeholder name="PagesListPage" /> },
      { path: '/pages/:slug',     element: <Placeholder name="PageEditorPage" /> },
      { path: '/blog',            element: <Placeholder name="BlogListPage" /> },
      { path: '/blog/new',        element: <Placeholder name="NewPostPage" /> },
      { path: '/blog/:id/edit',   element: <Placeholder name="EditPostPage" /> },
      { path: '/media',           element: <Placeholder name="MediaPage" /> },
      { path: '/forms',           element: <Placeholder name="FormsPage" /> },
      { path: '/settings',        element: <Placeholder name="SettingsPage" /> },
      { path: '/assistant',       element: <Placeholder name="AssistantPage" /> },
    ],
  },
])
