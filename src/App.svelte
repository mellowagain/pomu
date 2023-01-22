<script lang="ts">
    import "carbon-components-svelte/css/g100.css";
    import VideoInput from "./lib/VideoInput.svelte";
    import Logo from "./lib/Logo.svelte";
    import Heading from "./lib/Heading.svelte";
    import Notifications from "./lib/Notifications.svelte";
    import Nav from "./lib/Nav.svelte";
    import { Column, Content, Grid, Row } from "carbon-components-svelte";
    import { fade } from "svelte/transition";
    import VideoEmbed from "./lib/VideoEmbed.svelte";
    import Queue from "./lib/Queue.svelte";
    import History from "./lib/History.svelte";
    import { currentPage, Page } from "./lib/app";
    import Stats from "./lib/Stats.svelte";
</script>

<Notifications />
<Nav />
<Content>
    <div style="padding: 20px" />
    <Grid>
        <div transition:fade>
            {#if $currentPage === Page.Video}
                <Row>
                    <Column>
                        <Heading />
                        <Logo />
                        <Stats />
                    </Column>
                    <Column>
                        <VideoEmbed />
                        <VideoInput />
                    </Column>
                </Row>
            {:else if $currentPage === Page.Queue}
                <Queue />
            {:else if $currentPage === Page.History}
                <History />
            {/if}
        </div>
    </Grid>
</Content>
