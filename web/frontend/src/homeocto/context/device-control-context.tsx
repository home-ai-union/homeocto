import { createContext, useContext, type ReactNode } from "react"
import type { DeviceControlWSStatus } from "@/homeocto/api/device-control-websocket"
import type { CommunicationLog } from "@/homeocto/components/device-control-panel"
import type { ToolCallParams, ToolCallOptions } from "@/homeocto/api/device-command-executor"

interface DeviceControlContextValue {
  wsStatus: DeviceControlWSStatus
  logs: CommunicationLog[]
  showLogPanel: boolean
  logContainerRef: React.RefObject<HTMLDivElement | null>
  sendWebSocketMessage: (message: unknown) => void
  executeToolCall: (params: ToolCallParams, options?: ToolCallOptions) => Promise<{ success: boolean; error?: string }>
  clearLogs: () => void
  toggleLogPanel: () => void
  setShowLogPanel: (show: boolean) => void
  addLog: (log: Omit<CommunicationLog, "id" | "timestamp">) => void
}

const DeviceControlContext = createContext<DeviceControlContextValue | null>(null)

interface DeviceControlProviderProps {
  children: ReactNode
  value: DeviceControlContextValue
}

export function DeviceControlProvider({ children, value }: DeviceControlProviderProps) {
  return (
    <DeviceControlContext.Provider value={value}>
      {children}
    </DeviceControlContext.Provider>
  )
}

export function useDeviceControl() {
  const context = useContext(DeviceControlContext)
  if (!context) {
    throw new Error("useDeviceControl must be used within DeviceControlProvider")
  }
  return context
}
