import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface Notification {
    id: string
    type: 'info' | 'success' | 'warning' | 'error'
    title: string
    message: string
    time: string
    read: boolean
}

interface NotificationState {
    notifications: Notification[]
    addNotification: (notification: Omit<Notification, 'id' | 'time' | 'read'>) => void
    markAsRead: (id: string) => void
    markAllAsRead: () => void
    clearNotifications: () => void
}

const MAX_NOTIFICATIONS = 100

function formatNotificationTime() {
    const now = new Date()
    const pad = (value: number) => String(value).padStart(2, '0')
    return `${now.getFullYear()}/${pad(now.getMonth() + 1)}/${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
}

export const useNotificationStore = create<NotificationState>()(persist((set) => ({
    notifications: [],

    addNotification: (data) => set((state) => {
        const newNotification: Notification = {
            ...data,
            id: Math.random().toString(36).substring(2, 9),
            time: formatNotificationTime(),
            read: false,
        }
        return { notifications: [newNotification, ...state.notifications].slice(0, MAX_NOTIFICATIONS) }
    }),

    markAsRead: (id) => set((state) => ({
        notifications: state.notifications.map((n) =>
            n.id === id ? { ...n, read: true } : n
        ),
    })),

    markAllAsRead: () => set((state) => ({
        notifications: state.notifications.map((n) => ({ ...n, read: true })),
    })),

    clearNotifications: () => set({ notifications: [] }),
}), {
    name: 'ant-browser-notifications',
    partialize: (state) => ({ notifications: state.notifications }),
}))
