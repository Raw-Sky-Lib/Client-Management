import { createBrowserRouter, Navigate } from 'react-router'
import { GuestRoute } from '@/components/guards/GuestRoute'
import { ProtectedRoute } from '@/components/guards/ProtectedRoute'
import { PortalLayout } from '@/components/layout/portal-layout'
import { ConnectPage } from '@/features/onboarding/connect-page'
import { WelcomePage } from '@/features/onboarding/welcome-page'
import { LinkErrorPage } from '@/features/onboarding/link-error-page'
import { LoginPage } from '@/features/auth/login-page'
import { AuthCallbackPage } from '@/features/auth/auth-callback-page'

// Placeholders — replaced as each milestone builds the real component
const Placeholder = ({ name }: { name: string }) => (
  <div className="flex min-h-svh items-center justify-center bg-gray-50">
    <p className="text-sm text-gray-400">{name}</p>
  </div>
)

export const router = createBrowserRouter([
  {
    index: true,
    element: <Navigate to="/login" replace />,
  },

  // ─── Guest routes ─────────────────────────────────────────────────────────
  {
    element: <GuestRoute />,
    children: [
      { path: '/login',   element: <LoginPage /> },
      { path: '/connect', element: <ConnectPage /> },
    ],
  },

  // Auth callback — no guard; Supabase redirects here after magic link click
  {
    path: '/auth/callback',
    element: <AuthCallbackPage />,
  },

  // Link error — public, shown when a confirm link is used/expired/invalid
  {
    path: '/link-error',
    element: <LinkErrorPage />,
  },

  // ─── Protected routes ─────────────────────────────────────────────────────
  {
    element: <ProtectedRoute />,
    children: [
      // Welcome / onboarding-complete — own layout, no sidebar
      { path: '/welcome', element: <WelcomePage /> },

      {
        element: <PortalLayout />,
        children: [
          { path: '/dashboard',     element: <Placeholder name="DashboardPage" /> },
          { path: '/pages',         element: <Placeholder name="PagesListPage" /> },
          { path: '/pages/:slug',   element: <Placeholder name="PageEditorPage" /> },
          { path: '/blog',          element: <Placeholder name="BlogListPage" /> },
          { path: '/blog/new',      element: <Placeholder name="NewPostPage" /> },
          { path: '/blog/:id/edit', element: <Placeholder name="EditPostPage" /> },
          { path: '/media',         element: <Placeholder name="MediaPage" /> },
          { path: '/forms',         element: <Placeholder name="FormsPage" /> },
          { path: '/settings',      element: <Placeholder name="SettingsPage" /> },
          { path: '/assistant',     element: <Placeholder name="AssistantPage" /> },
        ],
      },
    ],
  },
])
