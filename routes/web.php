<?php

use App\Http\Controllers\Admin\AdminDashboardController;
use App\Http\Controllers\Admin\UserController as AdminUserController;
use App\Http\Controllers\Admin\LocationController as AdminLocationController;
use App\Http\Controllers\Admin\LocationSetupController;
use App\Http\Controllers\Admin\VortanixDaemonController;
use App\Http\Controllers\Admin\GameController as AdminGameController;
use App\Http\Controllers\Admin\TariffController as AdminTariffController;
use App\Http\Controllers\Admin\ServerController as AdminServerController;
use App\Http\Controllers\Admin\PluginController as AdminPluginController;
use App\Http\Controllers\AuthController;
use App\Http\Controllers\DashboardController;
use App\Http\Controllers\HomeController;
use App\Http\Controllers\AccountController;
use App\Http\Controllers\GamesController;
use App\Http\Controllers\FeaturesController;
use App\Http\Controllers\UserRentalController;
use App\Http\Controllers\UserBillingController;
use App\Http\Controllers\MonitoringController;
use App\Http\Controllers\ServerController;
use App\Http\Controllers\SupportController;
use App\Http\Controllers\LocaleController;
use App\Http\Controllers\PluginArchiveController;
use App\Http\Controllers\MapArchiveController;
use App\Http\Controllers\InstallController;
use App\Http\Controllers\Admin\MapController as AdminMapController;
use App\Http\Controllers\Admin\SettingsController as AdminSettingsController;
use App\Http\Controllers\Admin\UpdatesController as AdminUpdatesController;
use App\Http\Controllers\Admin\LicenseController as AdminLicenseController;
use App\Http\Controllers\Admin\BugReportController as AdminBugReportController;
use App\Http\Controllers\Admin\NotificationsController as AdminNotificationsController;
use App\Http\Controllers\Admin\NotificationsApiController as AdminNotificationsApiController;
use App\Http\Controllers\Admin\NewsController as AdminNewsController;
use App\Http\Controllers\Admin\SupportTicketController as AdminSupportTicketController;
use App\Http\Controllers\Admin\PromotionController as AdminPromotionController;
use App\Http\Controllers\Admin\LogsController as AdminLogsController;
use App\Http\Controllers\Admin\MailingController as AdminMailingController;
use App\Http\Controllers\Admin\LanguageController as AdminLanguageController;
use Illuminate\Support\Facades\Route;

Route::get('/install', [InstallController::class, 'show'])->name('install');
Route::post('/install', [InstallController::class, 'submit'])->name('install.submit');

Route::get('/', HomeController::class)->name('home');
Route::get('/games', [GamesController::class, 'index'])->name('games');
Route::get('/games/{slug}', [GamesController::class, 'show'])->name('games.show');
Route::get('/features', FeaturesController::class)->name('features');

Route::get('/plugins/{plugin}/download', [PluginArchiveController::class, 'download'])
    ->name('plugins.download')
    ->middleware('signed');

Route::get('/maps/{map}/download', [MapArchiveController::class, 'download'])
    ->name('maps.download')
    ->middleware('signed');

Route::middleware('guest')->group(function () {
    Route::get('/login', [AuthController::class, 'showLoginForm'])->name('login');
    Route::post('/login', [AuthController::class, 'login']);

    Route::get('/register', [AuthController::class, 'showRegisterForm'])->name('register');
    Route::post('/register', [AuthController::class, 'register']);
});

Route::middleware('auth')->group(function () {
    Route::get('/dashboard', [DashboardController::class, 'index'])->name('dashboard');
    Route::get('/account', [AccountController::class, 'show'])->name('account');
    Route::post('/locale/{locale}', [LocaleController::class, 'switch'])->name('locale.switch');
    Route::post('/account/profile', [AccountController::class, 'updateProfile'])->name('account.profile.update');
    Route::post('/account/email', [AccountController::class, 'updateEmail'])->name('account.email.update');
    Route::post('/account/password', [AccountController::class, 'updatePassword'])->name('account.password.update');
    Route::post('/account/sessions/destroy', [AccountController::class, 'destroySession'])->name('account.sessions.destroy');

    Route::get('/my-servers', [UserRentalController::class, 'myServers'])->name('my-servers');
    Route::get('/rent-server', [UserRentalController::class, 'rent'])->name('rent-server');
    Route::post('/rent-server', [UserRentalController::class, 'submitRent'])->name('rent-server.post');
    Route::get('/billing', [UserBillingController::class, 'index'])->name('billing');
    Route::get('/billing/topup', [UserBillingController::class, 'topup'])->name('billing.topup');
    Route::post('/billing/topup', [UserBillingController::class, 'createTopup'])->name('billing.topup.create');
    Route::get('/billing/topup/success', [UserBillingController::class, 'topupSuccess'])->name('billing.topup.success');
    Route::get('/billing/topup/fail', [UserBillingController::class, 'topupFail'])->name('billing.topup.fail');

    Route::get('/support', [SupportController::class, 'index'])->name('support.index');
    Route::get('/support/create', [SupportController::class, 'create'])->name('support.create');
    Route::post('/support', [SupportController::class, 'store'])->name('support.store');
    Route::get('/support/{ticket}', [SupportController::class, 'show'])->name('support.show');
    Route::get('/support/{ticket}/messages', [SupportController::class, 'messages'])->name('support.messages');
    Route::post('/support/{ticket}/reply', [SupportController::class, 'reply'])->name('support.reply');
    Route::post('/support/{ticket}/close', [SupportController::class, 'close'])->name('support.close');

    Route::get('/monitoring', [MonitoringController::class, 'index'])->name('monitoring');
    Route::post('/monitoring/data', [MonitoringController::class, 'data'])->name('monitoring.data');
    Route::get('/servers/{server}', [ServerController::class, 'show'])->name('server.show');
    Route::get('/servers/{server}/status', [ServerController::class, 'status'])->name('server.status');
    Route::get('/servers/{server}/console-logs', [ServerController::class, 'consoleLogs'])->name('server.console.logs');
    Route::post('/servers/{server}/console-command', [ServerController::class, 'consoleCommand'])->name('server.console.command');
    Route::get('/servers/{server}/metrics', [ServerController::class, 'metrics'])->name('server.metrics');
    Route::post('/servers/{server}/start', [ServerController::class, 'start'])->name('server.start');
    Route::post('/servers/{server}/stop', [ServerController::class, 'stop'])->name('server.stop');
    Route::post('/servers/{server}/restart', [ServerController::class, 'restart'])->name('server.restart');
    Route::post('/servers/{server}/reinstall', [ServerController::class, 'reinstall'])->name('server.reinstall');
    Route::post('/servers/{server}/auto-start', [ServerController::class, 'updateAutoStart'])->name('server.auto-start');
    Route::post('/servers/{server}/renew', [ServerController::class, 'renew'])->name('server.renew');
    Route::post('/servers/{server}/settings/samp', [ServerController::class, 'updateSampSettings'])->name('server.settings.samp');
    Route::post('/servers/{server}/settings/crmp', [ServerController::class, 'updateCrmpSettings'])->name('server.settings.crmp');
    Route::post('/servers/{server}/settings/rust', [ServerController::class, 'updateRustSettings'])->name('server.settings.rust');
    Route::post('/servers/{server}/settings/cs16', [ServerController::class, 'updateCs16Settings'])->name('server.settings.cs16');
    Route::post('/servers/{server}/settings/cs2', [ServerController::class, 'updateCs2Settings'])->name('server.settings.cs2');
    Route::post('/servers/{server}/settings/tf2', [ServerController::class, 'updateTf2Settings'])->name('server.settings.tf2');
    Route::post('/servers/{server}/settings/css', [ServerController::class, 'updateCssSettings'])->name('server.settings.css');
    Route::post('/servers/{server}/settings/gmod', [ServerController::class, 'updateGmodSettings'])->name('server.settings.gmod');
    Route::post('/servers/{server}/settings/unturned', [ServerController::class, 'updateUnturnedSettings'])->name('server.settings.unturned');
    Route::post('/servers/{server}/settings/minecraft', [ServerController::class, 'updateMinecraftSettings'])->name('server.settings.minecraft');
    Route::post('/servers/{server}/ftp/reset-password', [ServerController::class, 'resetFtpPassword'])->name('server.ftp.reset-password');
    Route::post('/servers/{server}/fastdl/configure', [ServerController::class, 'configureFastdl'])->name('server.fastdl.configure');
    Route::post('/servers/{server}/fastdl/update', [ServerController::class, 'updateFastdl'])->name('server.fastdl.update');
    Route::post('/servers/{server}/mysql/reset-password', [ServerController::class, 'resetMySqlPassword'])->name('server.mysql.reset-password');
    Route::post('/servers/{server}/files/list', [ServerController::class, 'filesList'])->name('server.files.list');
    Route::post('/servers/{server}/files/read', [ServerController::class, 'filesRead'])->name('server.files.read');
    Route::post('/servers/{server}/files/write', [ServerController::class, 'filesWrite'])->name('server.files.write');
    Route::post('/servers/{server}/files/mkdir', [ServerController::class, 'filesMkdir'])->name('server.files.mkdir');
    Route::post('/servers/{server}/files/delete', [ServerController::class, 'filesDelete'])->name('server.files.delete');
    Route::post('/servers/{server}/files/download', [ServerController::class, 'filesDownload'])->name('server.files.download');
    Route::post('/servers/{server}/files/upload', [ServerController::class, 'filesUpload'])->name('server.files.upload');

    Route::post('/servers/{server}/plugins/{plugin}/install', [ServerController::class, 'pluginInstall'])->name('server.plugins.install');
    Route::post('/servers/{server}/plugins/{plugin}/uninstall', [ServerController::class, 'pluginUninstall'])->name('server.plugins.uninstall');
    Route::post('/servers/{server}/plugins/{plugin}/toggle', [ServerController::class, 'pluginToggle'])->name('server.plugins.toggle');

    Route::post('/servers/{server}/maps/{map}/install', [ServerController::class, 'mapInstall'])->name('server.maps.install');
    Route::post('/servers/{server}/maps/{map}/uninstall', [ServerController::class, 'mapUninstall'])->name('server.maps.uninstall');

    Route::post('/servers/{server}/cron/list', [ServerController::class, 'cronList'])->name('server.cron.list');
    Route::post('/servers/{server}/cron/create', [ServerController::class, 'cronCreate'])->name('server.cron.create');
    Route::post('/servers/{server}/cron/delete', [ServerController::class, 'cronDelete'])->name('server.cron.delete');
    Route::post('/servers/{server}/cron/toggle', [ServerController::class, 'cronToggle'])->name('server.cron.toggle');

    Route::post('/servers/{server}/firewall/list', [ServerController::class, 'firewallList'])->name('server.firewall.list');
    Route::post('/servers/{server}/firewall/toggle', [ServerController::class, 'firewallToggle'])->name('server.firewall.toggle');
    Route::post('/servers/{server}/firewall/set', [ServerController::class, 'firewallSet'])->name('server.firewall.set');
    Route::post('/servers/{server}/firewall/add-rule', [ServerController::class, 'firewallAddRule'])->name('server.firewall.add-rule');
    Route::post('/servers/{server}/firewall/delete-rule', [ServerController::class, 'firewallDeleteRule'])->name('server.firewall.delete-rule');
    Route::post('/servers/{server}/firewall/toggle-rule', [ServerController::class, 'firewallToggleRule'])->name('server.firewall.toggle-rule');

    Route::post('/servers/{server}/friends/list', [ServerController::class, 'friendsList'])->name('server.friends.list');
    Route::post('/servers/{server}/friends/add', [ServerController::class, 'friendsAdd'])->name('server.friends.add');
    Route::post('/servers/{server}/friends/update', [ServerController::class, 'friendsUpdate'])->name('server.friends.update');
    Route::post('/servers/{server}/friends/delete', [ServerController::class, 'friendsDelete'])->name('server.friends.delete');

    Route::prefix('admin')->name('admin.')->group(function () {
        Route::get('/dashboard', [AdminDashboardController::class, 'index'])->name('dashboard');

        Route::get('/updates', [AdminUpdatesController::class, 'index'])->name('updates');
        Route::post('/updates/apply', [AdminUpdatesController::class, 'apply'])->name('updates.apply');

        Route::get('/license', [AdminLicenseController::class, 'index'])->name('license');
        Route::post('/license', [AdminLicenseController::class, 'store'])->name('license.store');
        Route::post('/license/clear', [AdminLicenseController::class, 'clear'])->name('license.clear');

        Route::get('/bug-report', [AdminBugReportController::class, 'index'])->name('bug-report');
        Route::post('/bug-report', [AdminBugReportController::class, 'store'])->name('bug-report.store');

        Route::get('/notifications', [AdminNotificationsController::class, 'index'])->name('notifications');
        Route::get('/api/notifications', [AdminNotificationsApiController::class, 'list'])->name('notifications.api.list');
        Route::post('/api/notifications/{id}/read', [AdminNotificationsApiController::class, 'read'])->name('notifications.api.read');

        Route::get('/logs', [AdminLogsController::class, 'index'])->name('logs.index');
        Route::get('/logs/tail', [AdminLogsController::class, 'tail'])->name('logs.tail');
        Route::get('/logs/download', [AdminLogsController::class, 'download'])->name('logs.download');

        Route::post('/mailings/{mailing}/start', [AdminMailingController::class, 'start'])->name('mailings.start');
        Route::post('/mailings/{mailing}/schedule', [AdminMailingController::class, 'schedule'])->name('mailings.schedule');
        Route::post('/mailings/{mailing}/cancel', [AdminMailingController::class, 'cancel'])->name('mailings.cancel');
        Route::resource('/mailings', AdminMailingController::class);

        Route::get('/language', [AdminLanguageController::class, 'edit'])->name('language.edit');
        Route::post('/language', [AdminLanguageController::class, 'update'])->name('language.update');

        Route::get('/settings', [AdminSettingsController::class, 'edit'])->name('settings.edit');
        Route::post('/settings', [AdminSettingsController::class, 'update'])->name('settings.update');

        Route::get('/users', [AdminUserController::class, 'index'])->name('users');
        Route::get('/users/create', [AdminUserController::class, 'create'])->name('users.create');
        Route::post('/users', [AdminUserController::class, 'store'])->name('users.store');
        Route::get('/users/{user}', [AdminUserController::class, 'show'])->name('users.show');
        Route::get('/users/{user}/edit', [AdminUserController::class, 'edit'])->name('users.edit');
        Route::put('/users/{user}', [AdminUserController::class, 'update'])->name('users.update');
        Route::delete('/users/{user}', [AdminUserController::class, 'destroy'])->name('users.destroy');

        Route::get('/support', [AdminSupportTicketController::class, 'index'])->name('support.index');
        Route::get('/support/{ticket}', [AdminSupportTicketController::class, 'show'])->name('support.show');
        Route::get('/support/{ticket}/messages', [AdminSupportTicketController::class, 'messages'])->name('support.messages');
        Route::post('/support/{ticket}/reply', [AdminSupportTicketController::class, 'reply'])->name('support.reply');
        Route::post('/support/{ticket}/close', [AdminSupportTicketController::class, 'close'])->name('support.close');
        Route::post('/support/{ticket}/reopen', [AdminSupportTicketController::class, 'reopen'])->name('support.reopen');
        Route::get('/locations', [AdminLocationController::class, 'index'])->name('locations.index');
        Route::get('/locations/create', [AdminLocationController::class, 'create'])->name('locations.create');
        Route::post('/locations', [AdminLocationController::class, 'store'])->name('locations.store');
        Route::get('/locations/{location}', [AdminLocationController::class, 'show'])->name('locations.show');
        Route::post('/locations/{location}/pull-daemon', [AdminLocationController::class, 'pullFromDaemon'])->name('locations.pull-daemon');
        Route::get('/locations/{location}/edit', [AdminLocationController::class, 'edit'])->name('locations.edit');
        Route::put('/locations/{location}', [AdminLocationController::class, 'update'])->name('locations.update');
        Route::patch('/locations/{location}/toggle', [AdminLocationController::class, 'toggle'])->name('locations.toggle');
        Route::delete('/locations/{location}', [AdminLocationController::class, 'destroy'])->name('locations.destroy');
        Route::get('/locations/{location}/setup', [LocationSetupController::class, 'index'])->name('locations.setup');
        Route::post('/locations/{location}/setup/packages', [LocationSetupController::class, 'installPackages'])->name('locations.setup.packages');
        Route::get('/locations/{location}/setup/status', [LocationSetupController::class, 'getStatus'])->name('locations.setup.status');
        Route::post('/locations/{location}/setup/docker', [LocationSetupController::class, 'installDocker'])->name('locations.setup.docker');
        Route::post('/locations/{location}/setup/mysql', [LocationSetupController::class, 'installMySQL'])->name('locations.setup.mysql');
        Route::post('/locations/{location}/setup/phpmyadmin', [LocationSetupController::class, 'installPhpMyAdmin'])->name('locations.setup.phpmyadmin');
        Route::post('/locations/{location}/setup/ftp', [LocationSetupController::class, 'installFTP'])->name('locations.setup.ftp');
        Route::post('/locations/{location}/setup/daemon', [LocationSetupController::class, 'installDaemon'])->name('locations.setup.daemon');
        Route::post('/locations/{location}/setup/images', [LocationSetupController::class, 'buildImages'])->name('locations.setup.images');
        Route::get('/vortanix-daemons', [VortanixDaemonController::class, 'index'])->name('vortanix-daemons.index');
        Route::get('/vortanix-daemons/{location}/daemon', [VortanixDaemonController::class, 'show'])->name('locations.daemon.show');
        Route::get('/vortanix-daemons/{location}/daemon/logs', [VortanixDaemonController::class, 'logs'])->name('locations.daemon.logs');
        Route::post('/vortanix-daemons/{location}/daemon/exec', [VortanixDaemonController::class, 'exec'])->name('locations.daemon.exec');
        Route::post('/locations/{location}/daemon/refresh', [VortanixDaemonController::class, 'refreshDaemon'])->name('locations.daemon.refresh');
        Route::post('/locations/{location}/daemon/restart', [VortanixDaemonController::class, 'restartDaemon'])->name('locations.daemon.restart');
        Route::get('/servers', [AdminServerController::class, 'index'])->name('servers.index');
        Route::get('/servers/{server}/manage', [ServerController::class, 'show'])->name('servers.manage');
        Route::post('/servers/{server}/reinstall', [AdminServerController::class, 'reinstall'])->name('servers.reinstall');
        Route::delete('/servers/{server}', [AdminServerController::class, 'destroy'])->name('servers.destroy');
        Route::get('/games', [AdminGameController::class, 'index'])->name('games');
        Route::get('/games/create', [AdminGameController::class, 'create'])->name('games.create');
        Route::post('/games', [AdminGameController::class, 'store'])->name('games.store');
        Route::get('/games/{game}', [AdminGameController::class, 'show'])->name('games.show');
        Route::get('/games/{game}/edit', [AdminGameController::class, 'edit'])->name('games.edit');
        Route::put('/games/{game}', [AdminGameController::class, 'update'])->name('games.update');
        Route::delete('/games/{game}', [AdminGameController::class, 'destroy'])->name('games.destroy');
        Route::post('/games/{game}/versions', [AdminGameController::class, 'storeVersion'])->name('games.versions.store');
        Route::delete('/games/{game}/versions/{version}', [AdminGameController::class, 'destroyVersion'])->name('games.versions.destroy');

        Route::resource('/plugins', AdminPluginController::class);

        Route::resource('/maps', AdminMapController::class);

        Route::resource('/news', AdminNewsController::class);

        Route::resource('/promotions', AdminPromotionController::class);

        Route::post('/tariffs/{tariff}/duplicate', [AdminTariffController::class, 'duplicate'])->name('tariffs.duplicate');
        Route::resource('/tariffs', AdminTariffController::class);
    });

    Route::post('/logout', [AuthController::class, 'logout'])->name('logout');
});
