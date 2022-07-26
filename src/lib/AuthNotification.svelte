<script lang="ts">
    import Notification from "./Notification.svelte";

    let params = new URL(document.location).searchParams;

    let displayRegister = params.has("successRegister");
    let displayLogin = params.has("success");
</script>

{#if displayRegister}
    <Notification title="Successfully registered" kind="success" />
{/if}

{#if displayLogin}
    <Notification title="Successfully logged in" kind="success" />
{/if}
