<script lang="ts">
    import { Popover, ToastNotification } from "carbon-components-svelte";
    import { createEventDispatcher } from "svelte";
    import { fade } from "svelte/transition";

    export let title;
    export let description = "";
    export let kind = "error";

    const dispatch = createEventDispatcher();
    function onClose(e) {
        dispatch("close", e);
    }
</script>

<ToastNotification
    lowContrast
    {title}
    subtitle={description}
    {kind}
    on:close={onClose}
/>

<style>
</style>
