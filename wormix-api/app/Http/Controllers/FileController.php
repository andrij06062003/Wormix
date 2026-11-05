<?php

namespace App\Http\Controllers;

use Illuminate\Support\Facades\File;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Response;

class FileController extends Controller
{
    public function getPhoto($photo, $name)
    {
//        Log::debug("Request photo", [
//            'storage' => $photo,
//            'name' => $name
//        ]);

        $full_path = "";
        switch (strtolower($photo)){
            case "bot":
                $full_path = resource_path("images/bots/{$name}");
                break;
            case "users":
                $full_path = resource_path("images/users/{$name}");
                break;
            default:
                $full_path == resource_path("images/default.png");
        }

        if(!File::exists($full_path))
            $full_path = resource_path("images/default.png");

        $file = File::get($full_path);
        $type = File::mimeType($full_path);
        $rsp = Response::make($file);
        $rsp->header('Cache-Control', 'no-transform,public,max-age=120');
        $rsp->header('Content-Type', $type);
        return $rsp;
    }
}
