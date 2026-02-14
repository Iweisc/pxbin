import {
  createRouter,
  createRootRoute,
  createRoute,
} from "@tanstack/react-router";
import { RootLayout } from "./routes/__root.tsx";
import { DashboardPage } from "./routes/index.tsx";
import { LoginPage } from "./routes/login.tsx";
import { LogsPage } from "./routes/logs.tsx";
import { CostsPage } from "./routes/costs.tsx";
import { KeysPage } from "./routes/keys.tsx";
import { ModelsPage } from "./routes/models.tsx";
import { UpstreamsPage } from "./routes/upstreams.tsx";

const rootRoute = createRootRoute({
  component: RootLayout,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: DashboardPage,
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  component: LoginPage,
});

const logsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/logs",
  component: LogsPage,
});

const costsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/costs",
  component: CostsPage,
});

const keysRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/keys",
  component: KeysPage,
});

const modelsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/models",
  component: ModelsPage,
});

const upstreamsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/upstreams",
  component: UpstreamsPage,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  logsRoute,
  costsRoute,
  keysRoute,
  modelsRoute,
  upstreamsRoute,
]);

export const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
