<script lang="ts">
    import "carbon-components-svelte/css/g100.css";
    import VideoInput from "./lib/VideoInput.svelte";
    import Logo from "./lib/Logo.svelte";
    import Heading from "./lib/Heading.svelte";
    import ArchiveButtons from "./lib/ArchiveButtons.svelte";
    import Notifications from "./lib/Notifications.svelte";
    import Nav from "./lib/Nav.svelte";
    import {
        Column,
        Content,
        Grid,
        Popover,
        Row,
    } from "carbon-components-svelte";
    import { fade } from "svelte/transition";
    import VideoEmbed from "./lib/VideoEmbed.svelte";
    import Queue from "./lib/Queue.svelte";
    import History from "./lib/History.svelte";
    import { currentPage, Page } from "./lib/app";

    let page = Page.Video;
    currentPage.subscribe((newPage) => (page = newPage));
</script>

<Notifications />
<Nav />
<Content>
    <div style="padding: 20px" />
    <Grid>
        <div transition:fade>
            {#if page == Page.Video}
                <Row>
                    <Column>
                        <Heading />
                        <Logo />
                    </Column>
                    <Column>
                        <VideoEmbed />
                        <VideoInput />
                    </Column>
                </Row>
            {:else if page == Page.Queue}
                <Queue />
            {:else if page == Page.History}
                <History />
            {/if}
        </div>
    </Grid>
</Content>
