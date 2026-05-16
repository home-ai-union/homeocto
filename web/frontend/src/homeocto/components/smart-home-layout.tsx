import { IconLoader2, IconAlertCircle, IconPlayerPlay } from "@tabler/icons-react"
import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { useGateway } from "@/hooks/use-gateway"

interface SmartHomeLayoutProps {
  /** Page title */
  title: string
  /** Children content */
  children: ReactNode
  /** Whether currently loading */
  isLoading?: boolean
}

/**
 * Simple layout wrapper for smart home pages.
 * WebSocket control panel is now managed globally in AppHeader.
 * Includes service status check - shows prompt page if service is not running.
 */
export function SmartHomeLayout({
  title,
  children,
  isLoading = false,
}: SmartHomeLayoutProps) {
  const { t } = useTranslation()
  const { state, loading, canStart, start } = useGateway()

  const isServiceRunning = state === "running"
  const isServiceLoading = loading || state === "starting"

  // Show service not ready page
  if (!isServiceRunning && !isServiceLoading) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-6 p-8">
        <div className="bg-muted/50 rounded-full p-6">
          <IconAlertCircle className="size-12 text-muted-foreground" />
        </div>

        <div className="text-center space-y-2 max-w-md">
          <h2 className="text-2xl font-semibold">
            {t("smartHome.serviceNotReady.title")}
          </h2>
          <p className="text-muted-foreground">
            {t("smartHome.serviceNotReady.description")}
          </p>
        </div>

        <Button
          onClick={() => void start()}
          disabled={loading || !canStart}
          size="lg"
          className="gap-2"
        >
          <IconPlayerPlay className="size-4" />
          {t("header.gateway.action.start")}
        </Button>
      </div>
    )
  }

  // Show loading state
  if (isServiceLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <IconLoader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // Original layout
  return (
    <div className="flex h-full flex-col">
      {/* Page header */}
      <div className="flex items-center gap-3 px-4 sm:px-6 pt-4 pb-2">
        <h1 className="text-xl font-semibold flex-1">{title}</h1>
        {isLoading && <IconLoader2 className="size-5 animate-spin text-muted-foreground" />}
      </div>

      {/* Page content (scrollable) */}
      <div className="min-h-0 flex-1 overflow-y-auto px-4 sm:px-6">
        {children}
      </div>
    </div>
  )
}
