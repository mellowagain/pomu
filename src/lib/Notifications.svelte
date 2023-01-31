<script lang="ts">
    import { NotificationActionButton } from "carbon-components-svelte";
    import { fade } from "svelte/transition";

    import Notification from "./Notification.svelte";
    import {
        notifications,
        clearNotification,
        StoredNotification,
        showNotification,
    } from "./notifications";

    let params = new URL(document.location).searchParams;

    if (params.has("success")) {
        showNotification({
            title: "Successfully Logged in",
            kind: "success",
            timeout: 5000,
            description: "",
        });
    }

    if (params.has("successRegister")) {
        showNotification({
            title: "Successfully Registered",
            kind: "success",
            timeout: 5000,
            description: "",
        });
    }

    if (params.has("successLogout")) {
        showNotification({
            title: "Successfully logged out",
            kind: "success",
            timeout: 5000,
            description: "",
        });
    }

    let notifs: [number, StoredNotification][];
    notifications.subscribe((n) => {
        notifs = [...n.entries()];
    });
</script>

<notifs>
    {#each notifs as [id, notification] (id)}
        <div transition:fade>
            <Notification
                on:close={(e) => {
                    clearNotification(id);
                }}
                title={notification.title}
                kind={notification.kind}
                description={notification.description}
            />
        </div>
    {/each}
</notifs>

<style>
    notifs {
        position: absolute;
        z-index: 6000;
        right: 10px;
        top: 40px;
    }
</style>
