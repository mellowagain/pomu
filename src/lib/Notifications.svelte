<script lang="ts">
    import { NotificationActionButton } from "carbon-components-svelte";
    import { fade } from "svelte/transition";

    import Notification from "./Notification.svelte";
    import { notifications, clearNotification } from "./notifications";

    let params = new URL(document.location).searchParams;

    let displayRegister = params.has("successRegister");
    let displayLogin = params.has("success");

    let notifs;
    notifications.subscribe((n) => {
        notifs = [...n.entries()];
    });
</script>

<notifs>
    {#if displayRegister}
        <Notification title="Successfully registered" kind="success" />
    {/if}

    {#if displayLogin}
        <Notification title="Successfully logged in" kind="success" />
    {/if}

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
