<script lang="ts">
    import {
        Header,
        HeaderAction,
        HeaderNavItem,
        HeaderPanelLink,
        HeaderUtilities,
        ImageLoader,
        SkipToContent,
    } from "carbon-components-svelte";
    import { currentPage, Page } from "./app";
    import { user } from "./api";
    import NavAvatar from "./NavAvatar.svelte";

    let isOpen = false;
</script>

<div>
    <Header
        company="Pomu.app"
        on:click={(_) => currentPage.update((_) => Page.Video)}
    >
        <svelte:fragment slot="skip-to-content">
            <SkipToContent />
        </svelte:fragment>
        <HeaderNavItem
            on:click={(_) => currentPage.update((_) => Page.Queue)}
            text="Queue"
        />
        <HeaderNavItem
            on:click={(_) => currentPage.update((_) => Page.History)}
            text="History"
        />
        <HeaderUtilities>
            {#await user then user}
                <HeaderAction icon={NavAvatar}>
                    <ImageLoader src={user.avatar} />
                    <HeaderPanelLink>{user.name}</HeaderPanelLink>
                </HeaderAction>
            {:catch e}
                <HeaderNavItem href="/login" text="Login" />
            {/await}
        </HeaderUtilities>
    </Header>
</div>

<style></style>
