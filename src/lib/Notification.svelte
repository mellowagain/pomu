<script lang="ts">
import {ToastNotification} from "carbon-components-svelte";

export let title;
export let description;
export let kind = "error";
</script>

<ToastNotification
        lowContrast
        title={title}
        subtitle={description}
        kind={kind}
/>
