<script lang="ts">
    import Countdown from "svelte-countdown/src";
    import dayjs from "dayjs";
    import { SkeletonText, Tag } from "carbon-components-svelte";
    import { Recording } from "carbon-icons-svelte";
    import { onMount } from "svelte";
    export let from: string;
</script>

<Countdown from={dayjs(from)} dateFormat="x" let:remaining>
    {#if !remaining.done}
        <p>
            Starts in
            {#if remaining.days > 0}
                <span>
                    {remaining.days + remaining.months * 30} day
                    {remaining.days === 1 ? "" : "s"}
                </span>
            {/if}

            {#if remaining.hours > 0}
                <span>
                    {remaining.hours} hour{remaining.hours === 1 ? "" : "s"}
                </span>
            {/if}

            <span>
                {remaining.minutes} minute{remaining.minute === 1 ? "" : "s"}
            </span>
            <span>
                {remaining.seconds} second{remaining.seconds === 1 ? "" : "s"}
            </span>
        </p>
    {:else}
        <Tag icon={Recording} type="red">Live</Tag>
    {/if}
</Countdown>
