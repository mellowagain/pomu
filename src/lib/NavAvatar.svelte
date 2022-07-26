<script lang="ts">
    import {
        HeaderAction,
        HeaderNavItem,
        ImageLoader,
    } from "carbon-components-svelte";

    import { user } from "./auth";
</script>

{#await user then user}
    <ImageLoader src={user.avatar} />
{/await}
