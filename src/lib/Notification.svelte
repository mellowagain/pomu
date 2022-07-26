<script lang="ts">
    import { Popover, ToastNotification } from "carbon-components-svelte";

    export let title;
    export let description = "";
    export let kind = "error";
</script>

<ToastNotification lowContrast {title} subtitle={description} {kind} />

<style>
</style>
