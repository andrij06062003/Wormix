<?php

namespace App\Modules;

use App\Models\User;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;

class UserSession
{
    private User $user;

    private array $user_session;

    /**
     * @throws \Exception
     */
    public function __construct(int $id)
    {
        $user = User::query()->where('id', $id)->get()->first();
        if($user == null)
            throw new \Exception("Can't find user with id {$id}");

        $this->user = $user;

        $session = null;
        try{
            $session = Cache::get('user_session_' . $user->id);
        }catch (\Exception $ex){
            Log::error("Session error: {$ex->getMessage()}");
            Cache::delete('user_session_' . $user->id);
        }
        if($session !== null) {
            $session = json_decode($session, true);
            if(@$session['id'] === null || @$session['auth_key'] === null || @$session['session_key'] === null || @$session['tcp_session'] === null)
                $this->user_session = $this->blankSession();
            else
                $this->user_session = $session;
        }
        else
            $this->user_session = $this->blankSession();
    }

    public function setAuthKey():string
    {
        $this->user_session['logged_in'] = false;
        $this->user_session['auth_key'] = hash("sha256", $this->user->id.openssl_random_pseudo_bytes(32));
        $this->user_session['session_key'] = '';
        Cache::set('user_session_' . $this->user->id, json_encode($this->user_session));
        return $this->user_session['auth_key'];
    }

    public function getAuthKey():string {
        return $this->user_session['auth_key'];
    }

    public function setSessionKey():string
    {
        $this->user_session['session_key'] = hash("sha256", $this->user_session['auth_key'].openssl_random_pseudo_bytes(32));
        Cache::set('user_session_' . $this->user->id, json_encode($this->user_session));
        return $this->user_session['session_key'];
    }

    public function getSessionKey():string {
        return $this->user_session['session_key'];
    }

    public function loggedIn(): void
    {
        $this->user_session['logged_in'] = true;
        Cache::set('user_session_' . $this->user->id, json_encode($this->user_session));
    }

    public function getSessionUser() : User
    {
        return $this->user;
    }

    public function isLoggedIn(): bool
    {
        return (bool)$this->user_session['logged_in'];
    }

    private function blankSession() : array
    {
        return [
            'id' => $this->user->id,
            'auth_key' => '',
            'session_key' => '',
            'tcp_session' => '',
            'logged_in' => false,
        ];
    }

    public function setTcpSession(string $session_id): void
    {
        $this->user_session['tcp_session'] = $session_id;
        Cache::set('user_session_' . $this->user->id, json_encode($this->user_session));
    }

    public function getTcpSession(): string
    {
        return $this->user_session['tcp_session'];
    }
}
