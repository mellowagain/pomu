<script lang="ts">
    import "carbon-components-svelte/css/g100.css";
    import Notifications from "./lib/Notifications.svelte";
    import Nav from "./lib/Nav.svelte";
    import { Content, Grid } from "carbon-components-svelte";
    import { fade } from "svelte/transition";
    import Queue from "./lib/Queue.svelte";
    import History from "./lib/History.svelte";
    import { currentPage, Page } from "./lib/app";
    import Video from "./lib/Video.svelte";
</script>

<Notifications />
<Nav />
<Content>
    <div style="padding: 20px" />
    <Grid>
        <div transition:fade>
            {#if $currentPage === Page.Video}
                <Video />
            {:else if $currentPage === Page.Queue}
                <Queue />
            {:else if $currentPage === Page.History}
                <History />
            {/if}
        </div>
    </Grid>
</Content>
