<?php
/**
 * Somente leitura no host CRM. Confere ID|plaintext, hash Sanctum, ability,
 * expiracao e policy de envio sem executar uma proposta real.
 * php verify-ctwa-sanctum-on-crm.php CRM_ROOT PRIVATE_TOKEN_FILE
 */
declare(strict_types=1);
if (PHP_SAPI !== 'cli' || $argc !== 3) {
    fwrite(STDERR, "USO: php verify-ctwa-sanctum-on-crm.php CRM_ROOT TOKEN_FILE_0600\n");
    exit(2);
}
$root = realpath($argv[1]);
$file = $argv[2];
if (!$root || !is_file($root . '/bootstrap/app.php') || !is_file($root . '/vendor/autoload.php') ||
    is_link($file) || !is_file($file) || (fileperms($file) & 0077) !== 0) {
    fwrite(STDERR, "NO_GO: host CRM/token privado ausentes ou inseguros.\n");
    exit(2);
}
try {
    require $root . '/vendor/autoload.php';
    $app = require $root . '/bootstrap/app.php';
    $app->make(\Illuminate\Contracts\Console\Kernel::class)->bootstrap();
    $plain = trim(file_get_contents($file));
    if (!preg_match('/^[1-9][0-9]*\|[a-zA-Z0-9]{40,}$/D', $plain)) {
        throw new RuntimeException('invalid format');
    }
    $token = \Laravel\Sanctum\PersonalAccessToken::findToken($plain);
    if (!$token || !$token->tokenable instanceof \App\Models\User ||
        !(bool)$token->tokenable->can_send_proposals ||
        !in_array(\App\Support\Extension\ExtensionAbilities::PROPOSAL_WRITE, (array)$token->abilities, true) ||
        count((array)$token->abilities) !== 1 ||
        ($token->expires_at !== null && $token->expires_at->isPast())) {
        throw new RuntimeException('invalid token/ability/user/expiry');
    }
    $expiryMinutes = config('sanctum.expiration');
    if ($expiryMinutes && $token->created_at->addMinutes((int)$expiryMinutes)->isPast()) {
        throw new RuntimeException('expired by global sanctum setting');
    }
    echo "PASS_SANCTUM_CRM: token existente, hash confirmado, ability minima, usuario autorizado, validade OK (segredo oculto)\n";
} catch (Throwable $ex) {
    fwrite(STDERR, "NO_GO_SANCTUM_CRM: validar identidade, ability ou expiracao no CRM (sem imprimir detalhes sensiveis).\n");
    exit(2);
}
