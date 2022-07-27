import { writable } from 'svelte/store'

interface StoredNotification {
    title: string;
    description: string;
    kind: string;
    timeout: number | null;
}

export let notifications = writable<Map<number, StoredNotification>>(new Map());

export function showNotification(notification: StoredNotification) {
    let id = Date.now();
    notifications.update(notifications => {
        notifications.set(id, notification);
        if (notification.timeout) {
            setTimeout(() => { clearNotification(id) }, notification.timeout)
        }
        return notifications;
    });
}
export function clearNotification(id: number) {
    console.log("clearing ", id);
    notifications.update(notifications => {
        notifications.delete(id);
        return notifications;
    })
}
