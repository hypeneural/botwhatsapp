<?php
/**
 * Execute SOMENTE no host real do Evydencia CRM, com acesso autorizado ao
 * Laravel e banco. Nao altera arquivos do repositorio CRM, apenas cria UM
 * PersonalAccessToken Sanctum para um usuario ja autorizado.
 *
 * php scripts/issue-ctwa-sanctum-on-crm.php /var/www/evydencia/current USER_ID /private/ctwa-sanctum.token
 * NAO enviar tokens em terminal, logs, issues ou chats; transferencia privada.
 */
declare(strict_types=1);
umask(0077);
if (PHP_SAPI !== 'cli' || $argc !== 4) {
    fwrite(STDERR, "USO: php issue-ctwa-sanctum-on-crm.php CRM_ROOT USER_ID OUTPUT_PRIVATE_0600\n");
    exit(2);
}
$crmRoot = realpath($argv[1]);
$userId = filter_var($argv[2], FILTER_VALIDATE_INT, ['options' => ['min_range' => 1]]);
$outFile = $argv[3];
if (!$crmRoot || !$userId ||
    !is_file($crmRoot . '/bootstrap/app.php') ||
    !is_file($crmRoot . '/vendor/autoload.php') ||
    file_exists($outFile) || is_link($outFile) || !is_dir(dirname($outFile))) {
    fwrite(STDERR, "NO_GO: CRM/root/usuario/arquivo privado invalido.\n");
    exit(2);
}
$parent = realpath(dirname($outFile));
if (!$parent || (fileperms($parent) & 0007) !== 0) {
    fwrite(STDERR, "NO_GO: diretorio privado com acesso world.\n");
    exit(2);
}
$created = null;
try {
    require $crmRoot . '/vendor/autoload.php';
    $app = require $crmRoot . '/bootstrap/app.php';
    $kernel = $app->make(\Illuminate\Contracts\Console\Kernel::class);
    $kernel->bootstrap();
    $user = \App\Models\User::query()->whereKey($userId)->first();
    if (!$user || !(bool) $user->can_send_proposals) {
        throw new RuntimeException('usuario sem permissao comercial no CRM');
    }
    $created = $user->createToken(
        'botwhatsapp-ctwa-5287-' . gmdate('Ymd'),
        [\App\Support\Extension\ExtensionAbilities::PROPOSAL_WRITE],
        now()->addDays(90)
    );
    $out = @fopen($outFile, 'x');
    if ($out === false) {
        throw new RuntimeException('arquivo ja existe ou nao pode ser criado');
    }
    @chmod($outFile, 0600);
    $plaintext = $created->plainTextToken;
    if (!is_string($plaintext) || !preg_match('/^[1-9][0-9]*\|[a-zA-Z0-9]{40,}$/D', $plaintext)) {
        fclose($out);
        @unlink($outFile);
        throw new RuntimeException('formato de token inesperado');
    }
    $written = fwrite($out, $plaintext . "\n");
    fflush($out);
    fclose($out);
    if ($written !== strlen($plaintext) + 1) {
        @unlink($outFile);
        throw new RuntimeException('arquivo de token incompleto');
    }
    $id = (int) $created->accessToken->getKey();
    fwrite(STDOUT, "TOKEN_ISSUED: id={$id} ability=extension:proposal-write expira_em_90_dias private_file=OK (valor oculto)\n");
    exit(0);
} catch (Throwable $e) {
    if ($created !== null) {
        try {
            $created->accessToken->delete();
        } catch (Throwable $ignore) {
        }
    }
    fwrite(STDERR, "NO_GO: emissao Sanctum bloqueada; revisar usuario/permissao/host, sem expor detalhes ou token.\n");
    exit(2);
}
