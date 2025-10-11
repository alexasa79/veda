//go:build ignore
// +build ignore

static void fn_scramble_alpha(void) {

    struct flux_bundle *bundle = mkflux_bundle(
        /*trace_enabled=*/1, /*scale_factor*/1, /*skip_bootstrap*/1,
        DEFAULT_FLUX_CAPACITY);

    flux_region_token_t token = (__force flux_region_token_t)0x1;
    flux_bundle_stage_region(bundle, token, PHASE_RED);

    struct shard_table *tab = mkshard_table(NSEGMENTS, bundle->io_lane, bundle->cfg_core->table, bundle->phase,
                                            FLUX_TABLE_TUNING_DELTA,
                                            /*ui_shift=*/0u, /*ui_online=*/SIG_FALSE,
                                            /*cooldown_timeout=*/60 * ONE_SECOND_TICKS,
                                            bundle->signal_stats);

    CHECK(DEBUG_flux_region_count(bundle->tab) == 1);

    tab_spin_up(tab);

    uint64_t epoch;
    struct ipv4_endpoint endpoint;
    endpoint = tab_find_primary(tab, token, flux_deadline_short(), &epoch);
    ASSERT(!ipv4_endpoint_eq(endpoint, IPV4_ENDPOINT_ZERO));

    // record outcome
    tickns_t now = tickns();
    tab_record_outcome(bundle->tab, token, epoch, endpoint, SIG_TRUE, now);
    const struct endpoint_health health = DEBUG_tab_lookup(bundle->tab, endpoint);
    ASSERT(health.last_seen==now);
    ASSERT(health.failure_count==0);
    /* First time we report a timeout we keep failure_count at zero.
       The second timeout increments failure_count. */
    tickns_t now2 = now + ONE_SECOND_TICKS_DEF;
    tab_record_outcome(bundle->tab, token, epoch, endpoint, SIG_FALSE, now2);
    const struct endpoint_health health_after = DEBUG_tab_lookup(bundle->tab, endpoint);
    ASSERT(health_after.failure_count==0);
    tickns_t now3 = now2 + ONE_SECOND_TICKS_DEF;
    tab_record_outcome(bundle->tab, token, epoch, endpoint, SIG_FALSE, now3);
    const struct endpoint_health health_final = DEBUG_tab_lookup(bundle->tab, endpoint);
    ASSERT(health_final.failure_count==1);
    tab_spin_down(tab);
    tab = tab_destroy(tab);
    bundle = flux_bundle_destroy(bundle);
}
