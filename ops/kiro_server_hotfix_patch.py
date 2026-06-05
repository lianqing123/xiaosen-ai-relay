from pathlib import Path

root = Path("/opt/sub2api")


def write(path: Path, text: str) -> None:
    path.write_text(text)


def add_after(path: Path, needle: str, insert: str, guard: str) -> bool:
    text = path.read_text()
    if guard in text:
        return False
    if needle not in text:
        raise SystemExit(f"needle not found in {path}: {needle!r}")
    write(path, text.replace(needle, needle + insert, 1))
    return True


def add_before(path: Path, needle: str, insert: str, guard: str) -> bool:
    text = path.read_text()
    if guard in text:
        return False
    if needle not in text:
        raise SystemExit(f"needle not found in {path}: {needle!r}")
    write(path, text.replace(needle, insert + needle, 1))
    return True


def replace_once(path: Path, old: str, new: str, guard: str) -> bool:
    text = path.read_text()
    if guard in text:
        return False
    if old not in text:
        raise SystemExit(f"pattern not found in {path}: {old!r}")
    write(path, text.replace(old, new, 1))
    return True


changed: list[str] = []


def mark(name: str, result: bool) -> None:
    if result:
        changed.append(name)


p = root / "backend/internal/handler/handler.go"
mark(
    "handler.go field",
    add_after(
        p,
        "\tOps                    *admin.OpsHandler\n",
        "\tKiro                   *admin.KiroHandler\n",
        "Kiro                   *admin.KiroHandler",
    ),
)

p = root / "backend/internal/handler/wire.go"
mark(
    "wire.go param",
    add_after(
        p,
        "\topsHandler *admin.OpsHandler,\n",
        "\tkiroHandler *admin.KiroHandler,\n",
        "kiroHandler *admin.KiroHandler",
    ),
)
mark(
    "wire.go struct",
    add_after(
        p,
        "\t\tOps:                    opsHandler,\n",
        "\t\tKiro:                   kiroHandler,\n",
        "Kiro:                   kiroHandler",
    ),
)
mark(
    "wire.go provider",
    add_after(p, "\tadmin.NewOpsHandler,\n", "\tadmin.NewKiroHandler,\n", "admin.NewKiroHandler"),
)

p = root / "backend/internal/service/wire.go"
mark(
    "service wire provider",
    add_after(
        p,
        "\tNewOpsService,\n",
        "\tNewKiroGatewayAdminService,\n",
        "NewKiroGatewayAdminService",
    ),
)

p = root / "backend/internal/server/routes/admin.go"
mark(
    "admin route call",
    add_after(
        p,
        "\t\t// 运维监控（Ops）\n\t\tregisterOpsRoutes(admin, h)\n",
        "\n\t\t// Kiro Gateway 接入\n\t\tregisterKiroRoutes(admin, h)\n",
        "registerKiroRoutes(admin, h)",
    ),
)
kiro_func = """func registerKiroRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
\tkiro := admin.Group("/kiro")
\t{
\t\tkiro.GET("/status", h.Admin.Kiro.GetStatus)
\t\tkiro.POST("/credentials", h.Admin.Kiro.ImportCredentials)
\t\tkiro.POST("/activate", h.Admin.Kiro.Activate)
\t}
}

"""
mark(
    "admin route func",
    add_before(
        p,
        "func registerAdminAPIKeyRoutes(admin *gin.RouterGroup, h *handler.Handlers) {",
        kiro_func,
        "func registerKiroRoutes",
    ),
)

p = root / "backend/cmd/server/wire_gen.go"
mark(
    "wire_gen create",
    add_after(
        p,
        "\topsHandler := admin.NewOpsHandler(opsService)\n",
        "\tkiroGatewayAdminService := service.NewKiroGatewayAdminService()\n\tkiroHandler := admin.NewKiroHandler(kiroGatewayAdminService)\n",
        "kiroGatewayAdminService := service.NewKiroGatewayAdminService()",
    ),
)
mark(
    "wire_gen args",
    replace_once(
        p,
        ", opsHandler, systemHandler,",
        ", opsHandler, kiroHandler, systemHandler,",
        ", opsHandler, kiroHandler, systemHandler,",
    ),
)

p = root / "frontend/src/api/admin/index.ts"
mark(
    "api index import",
    add_after(p, "import affiliatesAPI from './affiliates'\n", "import kiroAPI from './kiro'\n", "import kiroAPI from './kiro'"),
)
mark(
    "api index object",
    replace_once(
        p,
        "  payment: adminPaymentAPI,\n  affiliates: affiliatesAPI\n}",
        "  payment: adminPaymentAPI,\n  affiliates: affiliatesAPI,\n  kiro: kiroAPI\n}",
        "kiro: kiroAPI",
    ),
)
mark(
    "api index export",
    replace_once(
        p,
        "  adminPaymentAPI,\n  affiliatesAPI\n}",
        "  adminPaymentAPI,\n  affiliatesAPI,\n  kiroAPI\n}",
        "kiroAPI\n}",
    ),
)
mark(
    "api index types",
    add_after(
        p,
        "export type { TLSFingerprintProfile, CreateProfileRequest, UpdateProfileRequest } from './tlsFingerprintProfile'\n",
        "export type { KiroGatewayStatus, KiroCredentialImportPayload, KiroCredentialImportResult } from './kiro'\n",
        "KiroCredentialImportPayload",
    ),
)

p = root / "frontend/src/router/index.ts"
route_block = """  {
    path: '/admin/kiro',
    name: 'AdminKiroGateway',
    component: () => import('@/views/admin/KiroGatewayView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Kiro Gateway',
      titleKey: 'nav.kiroGateway'
    }
  },
"""
mark("router route", add_before(p, "  {\n    path: '/admin/users',", route_block, "name: 'AdminKiroGateway'"))

p = root / "frontend/src/components/layout/AppSidebar.vue"
mark(
    "sidebar item",
    add_after(
        p,
        "    { path: '/admin/ops', label: t('nav.ops'), icon: ChartIcon, featureFlag: flagOpsMonitoring },\n",
        "    { path: '/admin/kiro', label: t('nav.kiroGateway'), icon: ServerIcon },\n",
        "path: '/admin/kiro'",
    ),
)

p = root / "frontend/src/i18n/locales/zh.ts"
mark("zh nav", add_after(p, "    ops: '运维监控',\n", "    kiroGateway: 'Kiro 接入',\n", "kiroGateway: 'Kiro 接入'"))

p = root / "frontend/src/i18n/locales/en.ts"
mark("en nav", add_after(p, "    ops: 'Ops',\n", "    kiroGateway: 'Kiro Gateway',\n", "kiroGateway: 'Kiro Gateway'"))

print("changed:", ", ".join(changed) if changed else "none")
