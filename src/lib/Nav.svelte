<script lang="ts">
    import {
        Header,
        HeaderAction,
        HeaderGlobalAction,
        HeaderNavItem,
        HeaderPanelLink,
        HeaderUtilities,
        ImageLoader,
        SkipToContent,
    } from "carbon-components-svelte";
    import { currentPage, Page } from "./app";
    import { user } from "./api";
    import NavAvatar from "./NavAvatar.svelte";
    import { LogoGithub } from "carbon-icons-svelte";

    currentPage.subscribe(value => {
        switch (value) {
            case Page.Video:
                history.pushState({currentPage: value}, "", `${window.location.origin}`);
                break;
            case Page.Queue:
                history.pushState({currentPage: value}, "", `${window.location.origin}/queue`);
                break;
            case Page.History:
                history.pushState({currentPage: value}, "", `${window.location.origin}/history`);
                break;
            default:
                // TODO: Display 404
                break;
        }
    });

    window.onpopstate = function(event) {
        currentPage.set(event.state.currentPage);
        console.log(`location: ${document.location}, state: ${JSON.stringify(event.state)}`);
    }
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
            <HeaderGlobalAction
                aria-label="GitHub"
                icon={LogoGithub}
                on:click={() => window.open("https://github.com/mellowagain/pomu", "_blank")}
            />

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
