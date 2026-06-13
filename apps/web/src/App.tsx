import { createBrowserRouter, Navigate, Outlet, RouterProvider, ScrollRestoration } from "react-router";
import { PageTransition } from "@/components/PageTransition";
import { HomePage } from "@/pages/HomePage";
import { WaitlistPage } from "@/pages/WaitlistPage";
import { StorePage } from "@/pages/StorePage";
import { CollectionPage } from "@/pages/CollectionPage";
import { DesignPage } from "@/pages/DesignPage";
import { AboutPage } from "@/pages/AboutPage";
import { ContactPage } from "@/pages/ContactPage";
import { AccountPage } from "@/pages/AccountPage";
import { OrderDetailPage } from "@/pages/OrderDetailPage";
import { SlotsPage } from "@/pages/SlotsPage";
import { NotFoundPage } from "@/pages/NotFoundPage";
import { AdminLayout } from "@/pages/admin/AdminLayout";
import { AdminCollectionsPage } from "@/pages/admin/AdminCollectionsPage";
import { AdminDesignsPage } from "@/pages/admin/AdminDesignsPage";
import { AdminDesignEditorPage } from "@/pages/admin/AdminDesignEditorPage";
import { AdminOrdersPage } from "@/pages/admin/AdminOrdersPage";
import { AdminSubscribersPage } from "@/pages/admin/AdminSubscribersPage";
import { AdminAnalyticsPage } from "@/pages/admin/AdminAnalyticsPage";
import { AdminSettingsPage } from "@/pages/admin/AdminSettingsPage";
import { AdminSlotsPage } from "@/pages/admin/AdminSlotsPage";
import { AdminTeamPage } from "@/pages/admin/AdminTeamPage";
import { LoginPage } from "@/features/auth/LoginPage";
import { VerifyPage } from "@/features/auth/VerifyPage";
import { AuthGuard, AdminGuard, PermissionGuard } from "@/features/auth/guards";
import { PERMISSIONS } from "@/features/auth/permissions";

function RootLayout() {
  return (
    <>
      <ScrollRestoration />
      <PageTransition>
        <Outlet />
      </PageTransition>
    </>
  );
}

const router = createBrowserRouter([
  {
    element: <RootLayout />,
    children: [
      { path: "/", element: <HomePage /> },
  { path: "/waitlist", element: <WaitlistPage /> },
  { path: "/store", element: <StorePage /> },
  { path: "/collections/:slug", element: <CollectionPage /> },
  { path: "/designs/:slug", element: <DesignPage /> },
  { path: "/slots", element: <SlotsPage /> },
  { path: "/about", element: <AboutPage /> },
  { path: "/contact", element: <ContactPage /> },
  { path: "/login", element: <LoginPage /> },
  { path: "/auth/verify", element: <VerifyPage /> },
  {
    path: "/account",
    element: (
      <AuthGuard>
        <AccountPage />
      </AuthGuard>
    ),
  },
  {
    path: "/account/orders/:ref",
    element: (
      <AuthGuard>
        <OrderDetailPage />
      </AuthGuard>
    ),
  },
  {
    path: "/admin",
    element: (
      <AdminGuard>
        <AdminLayout />
      </AdminGuard>
    ),
    children: [
      { index: true, element: <Navigate to="/admin/designs" replace /> },
      { path: "designs", element: <AdminDesignsPage /> },
      {
        path: "designs/new",
        element: (
          <PermissionGuard permission={PERMISSIONS.catalogueWrite}>
            <AdminDesignEditorPage />
          </PermissionGuard>
        ),
      },
      {
        path: "designs/:id",
        element: (
          <PermissionGuard permission={PERMISSIONS.catalogueWrite}>
            <AdminDesignEditorPage />
          </PermissionGuard>
        ),
      },
      { path: "collections", element: <AdminCollectionsPage /> },
      { path: "orders", element: <AdminOrdersPage /> },
      { path: "subscribers", element: <AdminSubscribersPage /> },
      { path: "slots", element: <AdminSlotsPage /> },
      { path: "analytics", element: <AdminAnalyticsPage /> },
      {
        path: "team",
        element: (
          <PermissionGuard permission={PERMISSIONS.teamRead}>
            <AdminTeamPage />
          </PermissionGuard>
        ),
      },
      {
        path: "settings",
        element: (
          <PermissionGuard permission={PERMISSIONS.settingsWrite}>
            <AdminSettingsPage />
          </PermissionGuard>
        ),
      },
    ],
  },
      { path: "*", element: <NotFoundPage /> },
    ],
  },
]);

export default function App() {
  return <RouterProvider router={router} />;
}
