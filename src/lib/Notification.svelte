<script lang="ts">
import {ToastNotification} from "carbon-components-svelte";

export let title;
export let description = "";
export let kind = "error";
</script>

<div>
    <ToastNotification
            lowContrast
            title={title}
            subtitle={description}
            kind={kind}
    />
</div>

<style>
    div {
        position: absolute;
        top: 0;
        right: 0;
    }
</style>
