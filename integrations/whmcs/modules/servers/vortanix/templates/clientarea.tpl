<div class="vortanix-service">
    {if $vtxError}
        <div class="alert alert-warning">{$vtxLang.client_unavailable|escape}</div>
    {elseif !$vtxServer}
        <div class="alert alert-info">{$vtxLang.server_missing|escape}</div>
    {else}
        {if $vtxSuspended}
            <div class="alert alert-danger">{$vtxLang.client_suspended|escape}</div>
        {elseif $vtxBlocked}
            <div class="alert alert-danger">{$vtxBlocked|escape}</div>
        {/if}
        <div class="row">
            <div class="col-md-7">
                <h4>{$vtxServer.name|escape}</h4>
                <table class="table table-condensed table-sm">
                    <tbody>
                        <tr>
                            <td>{$vtxLang.client_status|escape}</td>
                            <td>
                                <span class="label label-{$vtxStatusTone|escape} badge badge-{$vtxStatusBadge|escape}">{$vtxStatusLabel|escape}</span>
                            </td>
                        </tr>
                        {foreach from=$vtxRows item=row}
                            <tr>
                                <td>{$row.label|escape}</td>
                                <td>
                                    {if $row.code}
                                        <code>{$row.value|escape}</code>
                                    {else}
                                        {$row.value|escape}
                                    {/if}
                                </td>
                            </tr>
                        {/foreach}
                    </tbody>
                </table>
            </div>
            <div class="col-md-5">
                <p>
                    <a class="btn btn-primary btn-block" href="clientarea.php?action=productdetails&amp;id={$vtxServiceId}&amp;dosinglesignon=1" target="_blank" rel="noopener">{$vtxLang.client_open_panel|escape}</a>
                </p>
                {if $vtxCanPower}
                    <p>
                        <a class="btn btn-default btn-outline-secondary btn-block" href="clientarea.php?action=productdetails&amp;id={$vtxServiceId}&amp;modop=custom&amp;a=start">{$vtxLang.button_start|escape}</a>
                        <a class="btn btn-default btn-outline-secondary btn-block" href="clientarea.php?action=productdetails&amp;id={$vtxServiceId}&amp;modop=custom&amp;a=restart">{$vtxLang.button_restart|escape}</a>
                        <a class="btn btn-default btn-outline-secondary btn-block" href="clientarea.php?action=productdetails&amp;id={$vtxServiceId}&amp;modop=custom&amp;a=stop">{$vtxLang.button_stop|escape}</a>
                    </p>
                {/if}
            </div>
        </div>
    {/if}
</div>
