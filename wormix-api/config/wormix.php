<?php
return [
    'starter' => [
        'money' => 450,
        'real_money' => 3,
        'missions' => 10,
        'race' => 0,
        'weapons' => [1, 2, 4]
    ],
    'game' => [
        'missions' => [
            'delay' => 120,
            'max' => 5,
            'awards' => [

                'loose' => [
                    'money' => 5,
                    'experience' => 3
                ],

                'draw' => [
                    'money' => [
                        'low' => 20,
                        'medium' => 25,
                        'high' => 30
                    ],
                    'experience' => [
                        'low' => 4,
                        'medium' => 6,
                        'high' => 8
                    ]
                ],

                'win' => [
                    'money' => [
                        'low' => 30,
                        'medium' => 35,
                        'high' => 40
                    ],
                    'experience' => [
                        'low' => 8,
                        'medium' => 10,
                        'high' => 12
                    ]
                ]
            ],

            'buy' => [
                'money' => 100,
                'real_money' => 1,
            ]
        ],
        'buy' => [
            'boss_mission' => 10,
            'reset_stats' => [
                'money' => 300,
                'real_money' => 3,
            ],
            'teammate' => [
                'money' => 950,
                'real_money' => 10,
            ],
            'downgrade' => 5
        ],
        'search_keys_per_day' => 10,
        'next_level_award' => [
            'money' => 150,
            'real_money' => 0,
        ]
    ],
    'vk_balance' => 10
];
