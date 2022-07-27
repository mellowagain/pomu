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
    import VideoEmbed from "./lib/VideoEmbed.svelte";
</script>

<Notifications />
<Nav />
<Content>
    <div style="padding: 20px" />
    <Grid>
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
    </Grid>
</Content>
