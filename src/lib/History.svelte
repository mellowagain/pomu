<script lang="ts">
    import {
        Column,
        Dropdown,
        Grid,
        InlineNotification,
        NotificationActionButton,
        Pagination,
        PaginationSkeleton,
        Row,
        Toggle
    } from "carbon-components-svelte";
    import SkeletonVideoEntry from "./SkeletonVideoEntry.svelte";
    import type { HistoryResponse } from "./video";
    import VideoEntry from "./VideoEntry.svelte";
    import { onDestroy, onMount } from "svelte";

    // Search parameters
    let sorting = "desc";
    let displayUnfinished = false;
    let page = 1;
    let limit = 25;

    let history = requestHistory();
    let intervalId;

    function refreshData(invokedByInterval = false) {
        history = requestHistory();

        // If we manually refresh the data, reschedule the interval, so it doesn't cause double refreshes
        if (!invokedByInterval) {
            try {
                clearInterval(intervalId);
            } catch {
                // In case the interval is already cleared, clearInterval will throw an error but that's fine with us so just continue
            }

            scheduleRefresh();
        }
    }

    async function requestHistory(): Promise<HistoryResponse> {
        let results = await fetch(`/api/history?page=${page - 1}&limit=${limit}&sort=${sorting}&unfinished=${displayUnfinished}`);
        let json = await results.json();

        return {
            totalItems: +results.headers.get("X-Pomu-Pagination-Total"),
            videos: new Map(json.map(entry => [entry.id, entry]))
        };
    }

    function scheduleRefresh() {
        intervalId = setInterval(() => refreshData(true), 60000);
    }

    function handleVisibilityChange() {
        switch (document.visibilityState) {
            case "visible":
                refreshData();
                scheduleRefresh();
                break;
            case "hidden":
                clearInterval(intervalId);
                break;
        }
    }

    onMount(() => scheduleRefresh());
    onDestroy(() => clearInterval(intervalId));
</script>

<svelte:window on:visibilitychange={handleVisibilityChange}/>

<Grid>
    <Row>
        <Column>
            <h1>History</h1>
        </Column>
        <Column></Column>
        <Column>
            <div class="toggler">
                <Toggle
                    labelText="Display unfinished"
                    bind:toggled={displayUnfinished}
                    on:toggle={refreshData}
                />
            </div>
        </Column>
        <Column>
            <Dropdown
                titleText="Sorting"
                bind:selectedId={sorting}
                items={[
                    { id: "asc", text: "Oldest" },
                    { id: "desc", text: "Newest" }
                ]}
                on:select={refreshData}
            />
        </Column>
    </Row>
</Grid>

<div class="divider"></div>

{#await history}
    <PaginationSkeleton />

    {#each Array(limit) as _, i}
        <SkeletonVideoEntry />
    {/each}

    <PaginationSkeleton />
{:then result}
    {#if result.videos.size === 0}
        <InlineNotification
            lowContrast
            hideCloseButton
            kind="info"
            subtitle="No streams have finished recording"
            on:close={(e) => {
                e.preventDefault();
            }}
        />
    {:else}
        <Pagination
            totalItems={result.totalItems}
            pageSizes={[25, 50, 75, 100]}
            bind:pageSize={limit}
            bind:page
            on:click:button--previous={refreshData}
            on:click:button--next={refreshData}
        />

        {#each [...result.videos.entries()] as [id, info] (id)}
            <VideoEntry {info} />
        {/each}

        <Pagination
            totalItems={result.totalItems}
            pageSizes={[25, 50, 75, 100]}
            bind:pageSize={limit}
            bind:page
            on:update={refreshData}
            on:click:button--previous={refreshData}
            on:click:button--next={refreshData}
        />
    {/if}
{:catch error}
    <InlineNotification
        lowContrast
        hideCloseButton
        kind="error"
        title="Failed to load history:"
        subtitle={error}
    >
        <svelte:fragment slot="actions">
            <NotificationActionButton on:click={refreshData}>Refresh</NotificationActionButton>
        </svelte:fragment>
    </InlineNotification>
{/await}

<style>
    .toggler {
        float: right;
    }

    .divider {
        margin-bottom: 1em;
    }
</style>
