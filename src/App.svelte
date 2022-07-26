<script lang="ts">
    import "carbon-components-svelte/css/g100.css";
    import VideoInput from "./lib/VideoInput.svelte";
    import Logo from "./lib/Logo.svelte";
    import Heading from "./lib/Heading.svelte";
    import ArchiveButtons from "./lib/ArchiveButtons.svelte";
    import AuthNotification from "./lib/AuthNotification.svelte";
    import Nav from "./lib/Nav.svelte";
    import {
        Column,
        Content,
        Grid,
        Popover,
        Row,
    } from "carbon-components-svelte";
</script>

<Popover open align="top-right" style="top: 40px">
    <AuthNotification />
</Popover>
<Nav />
<Content>
    <div style="padding: 20px" />
    <Grid>
        <Row>
            <Column>
                <Heading />
                <Logo />
            </Column>
        </Row>
        <Row>
            <Column>
                <VideoInput />
            </Column>
        </Row>
    </Grid>
</Content>

<style>
    main {
        display: flex;
        align-items: center;
        justify-content: center;
    }
</style>
