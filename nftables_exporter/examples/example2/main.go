package main

import (
	"fmt"
	"github.com/tidwall/gjson"
)

func main() {
	jsonData := `{
    "nftables": [
        {
            "metainfo": {
                "version": "1.0.9",
                "release_name": "Old Doc Yak #3",
                "json_schema_version": 1
            }
        },
        {
            "table": {
                "family": "inet",
                "name": "lxc",
                "handle": 11
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "lxc",
                "name": "input",
                "handle": 1,
                "type": "filter",
                "hook": "input",
                "prio": 0,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "lxc",
                "name": "forward",
                "handle": 6,
                "type": "filter",
                "hook": "forward",
                "prio": 0,
                "policy": "accept"
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "input",
                "handle": 3,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "udp",
                                    "field": "dport"
                                }
                            },
                            "right": {
                                "set": [
                                    53,
                                    67
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "input",
                "handle": 5,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "tcp",
                                    "field": "dport"
                                }
                            },
                            "right": {
                                "set": [
                                    53,
                                    67
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "forward",
                "handle": 7,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "forward",
                "handle": 8,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "oifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "table": {
                "family": "ip",
                "name": "lxc",
                "handle": 12
            }
        },
        {
            "chain": {
                "family": "ip",
                "table": "lxc",
                "name": "postrouting",
                "handle": 1,
                "type": "nat",
                "hook": "postrouting",
                "prio": 100,
                "policy": "accept"
            }
        },
        {
            "rule": {
                "family": "ip",
                "table": "lxc",
                "chain": "postrouting",
                "handle": 2,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "ip",
                                    "field": "saddr"
                                }
                            },
                            "right": {
                                "prefix": {
                                    "addr": "10.0.3.0",
                                    "len": 24
                                }
                            }
                        }
                    },
                    {
                        "match": {
                            "op": "!=",
                            "left": {
                                "payload": {
                                    "protocol": "ip",
                                    "field": "daddr"
                                }
                            },
                            "right": {
                                "prefix": {
                                    "addr": "10.0.3.0",
                                    "len": 24
                                }
                            }
                        }
                    },
                    {
                        "counter": {
                            "packets": 3,
                            "bytes": 335
                        }
                    },
                    {
                        "masquerade": null
                    }
                ]
            }
        }
    ]
}`
	jsonData = `{
    "nftables": [
        {
            "metainfo": {
                "version": "1.0.9",
                "release_name": "Old Doc Yak #3",
                "json_schema_version": 1
            }
        },
        {
            "table": {
                "family": "inet",
                "name": "lxc",
                "handle": 11
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "lxc",
                "name": "input",
                "handle": 1,
                "type": "filter",
                "hook": "input",
                "prio": 0,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "lxc",
                "name": "forward",
                "handle": 6,
                "type": "filter",
                "hook": "forward",
                "prio": 0,
                "policy": "accept"
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "input",
                "handle": 3,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "udp",
                                    "field": "dport"
                                }
                            },
                            "right": {
                                "set": [
                                    53,
                                    67
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "input",
                "handle": 5,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "tcp",
                                    "field": "dport"
                                }
                            },
                            "right": {
                                "set": [
                                    53,
                                    67
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "forward",
                "handle": 7,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "forward",
                "handle": 8,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "oifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "table": {
                "family": "ip",
                "name": "lxc",
                "handle": 12
            }
        },
        {
            "chain": {
                "family": "ip",
                "table": "lxc",
                "name": "postrouting",
                "handle": 1,
                "type": "nat",
                "hook": "postrouting",
                "prio": 100,
                "policy": "accept"
            }
        },
        {
            "rule": {
                "family": "ip",
                "table": "lxc",
                "chain": "postrouting",
                "handle": 2,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "ip",
                                    "field": "saddr"
                                }
                            },
                            "right": {
                                "prefix": {
                                    "addr": "10.0.3.0",
                                    "len": 24
                                }
                            }
                        }
                    },
                    {
                        "match": {
                            "op": "!=",
                            "left": {
                                "payload": {
                                    "protocol": "ip",
                                    "field": "daddr"
                                }
                            },
                            "right": {
                                "prefix": {
                                    "addr": "10.0.3.0",
                                    "len": 24
                                }
                            }
                        }
                    },
                    {
                        "counter": {
                            "packets": 3,
                            "bytes": 335
                        }
                    },
                    {
                        "masquerade": null
                    }
                ]
            }
        },
        {
            "table": {
                "family": "inet",
                "name": "firewalld",
                "handle": 14
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PREROUTING",
                "handle": 139,
                "type": "filter",
                "hook": "prerouting",
                "prio": -140,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PREROUTING_POLICIES_pre",
                "handle": 140
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PREROUTING_ZONES",
                "handle": 141
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PREROUTING_POLICIES_post",
                "handle": 142
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PREROUTING",
                "handle": 144,
                "type": "nat",
                "hook": "prerouting",
                "prio": -90,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PREROUTING_POLICIES_pre",
                "handle": 145
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PREROUTING_ZONES",
                "handle": 146
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PREROUTING_POLICIES_post",
                "handle": 147
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POSTROUTING",
                "handle": 149,
                "type": "nat",
                "hook": "postrouting",
                "prio": 110,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POSTROUTING_POLICIES_pre",
                "handle": 150
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POSTROUTING_ZONES",
                "handle": 151
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POSTROUTING_POLICIES_post",
                "handle": 152
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_OUTPUT",
                "handle": 154,
                "type": "nat",
                "hook": "output",
                "prio": -90,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_OUTPUT_POLICIES_pre",
                "handle": 155
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_OUTPUT_POLICIES_post",
                "handle": 157
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_PREROUTING",
                "handle": 159,
                "type": "filter",
                "hook": "prerouting",
                "prio": 10,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_INPUT",
                "handle": 160,
                "type": "filter",
                "hook": "input",
                "prio": 10,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FORWARD",
                "handle": 161,
                "type": "filter",
                "hook": "forward",
                "prio": 10,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_OUTPUT",
                "handle": 162,
                "type": "filter",
                "hook": "output",
                "prio": 10,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_INPUT_POLICIES_pre",
                "handle": 168
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_INPUT_ZONES",
                "handle": 169
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_INPUT_POLICIES_post",
                "handle": 170
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FORWARD_POLICIES_pre",
                "handle": 178
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FORWARD_ZONES",
                "handle": 179
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FORWARD_POLICIES_post",
                "handle": 181
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_OUTPUT_POLICIES_pre",
                "handle": 186
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_OUTPUT_POLICIES_post",
                "handle": 188
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_public",
                "handle": 197
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_public_pre",
                "handle": 198
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_public_log",
                "handle": 199
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_public_deny",
                "handle": 200
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_public_allow",
                "handle": 201
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_public_post",
                "handle": 202
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POST_public",
                "handle": 214
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POST_public_pre",
                "handle": 215
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POST_public_log",
                "handle": 216
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POST_public_deny",
                "handle": 217
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POST_public_allow",
                "handle": 218
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_POST_public_post",
                "handle": 219
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FWD_public",
                "handle": 227
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FWD_public_pre",
                "handle": 228
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FWD_public_log",
                "handle": 229
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FWD_public_deny",
                "handle": 230
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FWD_public_allow",
                "handle": 231
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_FWD_public_post",
                "handle": 232
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_public",
                "handle": 241
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_public_pre",
                "handle": 242
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_public_log",
                "handle": 243
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_public_deny",
                "handle": 244
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_public_allow",
                "handle": 245
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_public_post",
                "handle": 246
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_public",
                "handle": 254
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_public_pre",
                "handle": 255
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_public_log",
                "handle": 256
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_public_deny",
                "handle": 257
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_public_allow",
                "handle": 258
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_public_post",
                "handle": 259
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_policy_allow-host-ipv6",
                "handle": 274
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_policy_allow-host-ipv6_pre",
                "handle": 275
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_policy_allow-host-ipv6_log",
                "handle": 276
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_policy_allow-host-ipv6_deny",
                "handle": 277
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_policy_allow-host-ipv6_allow",
                "handle": 278
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "filter_IN_policy_allow-host-ipv6_post",
                "handle": 279
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_policy_allow-host-ipv6",
                "handle": 285
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_policy_allow-host-ipv6_pre",
                "handle": 286
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_policy_allow-host-ipv6_log",
                "handle": 287
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_policy_allow-host-ipv6_deny",
                "handle": 288
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_policy_allow-host-ipv6_allow",
                "handle": 289
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "nat_PRE_policy_allow-host-ipv6_post",
                "handle": 290
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_policy_allow-host-ipv6",
                "handle": 296
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_policy_allow-host-ipv6_pre",
                "handle": 297
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_policy_allow-host-ipv6_log",
                "handle": 298
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_policy_allow-host-ipv6_deny",
                "handle": 299
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_policy_allow-host-ipv6_allow",
                "handle": 300
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "firewalld",
                "name": "mangle_PRE_policy_allow-host-ipv6_post",
                "handle": 301
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PREROUTING",
                "handle": 143,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PREROUTING_ZONES"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PREROUTING_POLICIES_pre",
                "handle": 309,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_policy_allow-host-ipv6"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PREROUTING_ZONES",
                "handle": 318,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "eth0"
                        }
                    },
                    {
                        "goto": {
                            "target": "mangle_PRE_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PREROUTING_ZONES",
                "handle": 273,
                "expr": [
                    {
                        "goto": {
                            "target": "mangle_PRE_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PREROUTING",
                "handle": 148,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PREROUTING_ZONES"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PREROUTING_POLICIES_pre",
                "handle": 308,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_policy_allow-host-ipv6"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PREROUTING_ZONES",
                "handle": 317,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "eth0"
                        }
                    },
                    {
                        "goto": {
                            "target": "nat_PRE_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PREROUTING_ZONES",
                "handle": 272,
                "expr": [
                    {
                        "goto": {
                            "target": "nat_PRE_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POSTROUTING",
                "handle": 153,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_POSTROUTING_ZONES"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POSTROUTING_ZONES",
                "handle": 315,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "oifname"
                                }
                            },
                            "right": "eth0"
                        }
                    },
                    {
                        "goto": {
                            "target": "nat_POST_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POSTROUTING_ZONES",
                "handle": 270,
                "expr": [
                    {
                        "goto": {
                            "target": "nat_POST_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_OUTPUT",
                "handle": 156,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_OUTPUT_POLICIES_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_OUTPUT",
                "handle": 158,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_OUTPUT_POLICIES_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_PREROUTING",
                "handle": 192,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "icmpv6",
                                    "field": "type"
                                }
                            },
                            "right": {
                                "set": [
                                    "nd-router-advert",
                                    "nd-neighbor-solicit"
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_PREROUTING",
                "handle": 190,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "nfproto"
                                }
                            },
                            "right": "ipv6"
                        }
                    },
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "fib": {
                                    "result": "oif",
                                    "flags": [
                                        "saddr",
                                        "mark",
                                        "iif"
                                    ]
                                }
                            },
                            "right": false
                        }
                    },
                    {
                        "drop": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_INPUT",
                "handle": 164,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "ct": {
                                    "key": "state"
                                }
                            },
                            "right": {
                                "set": [
                                    "established",
                                    "related"
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_INPUT",
                "handle": 165,
                "expr": [
                    {
                        "match": {
                            "op": "in",
                            "left": {
                                "ct": {
                                    "key": "status"
                                }
                            },
                            "right": "dnat"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_INPUT",
                "handle": 166,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lo"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_INPUT",
                "handle": 167,
                "expr": [
                    {
                        "match": {
                            "op": "in",
                            "left": {
                                "ct": {
                                    "key": "state"
                                }
                            },
                            "right": "invalid"
                        }
                    },
                    {
                        "drop": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_INPUT",
                "handle": 171,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_INPUT_ZONES"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_INPUT",
                "handle": 172,
                "expr": [
                    {
                        "reject": {
                            "type": "icmpx",
                            "expr": "admin-prohibited"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FORWARD",
                "handle": 174,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "ct": {
                                    "key": "state"
                                }
                            },
                            "right": {
                                "set": [
                                    "established",
                                    "related"
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FORWARD",
                "handle": 175,
                "expr": [
                    {
                        "match": {
                            "op": "in",
                            "left": {
                                "ct": {
                                    "key": "status"
                                }
                            },
                            "right": "dnat"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FORWARD",
                "handle": 176,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lo"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FORWARD",
                "handle": 177,
                "expr": [
                    {
                        "match": {
                            "op": "in",
                            "left": {
                                "ct": {
                                    "key": "state"
                                }
                            },
                            "right": "invalid"
                        }
                    },
                    {
                        "drop": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FORWARD",
                "handle": 196,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "ip6",
                                    "field": "daddr"
                                }
                            },
                            "right": {
                                "set": [
                                    {
                                        "prefix": {
                                            "addr": "::",
                                            "len": 96
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "::ffff:0.0.0.0",
                                            "len": 96
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002::",
                                            "len": 24
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:a00::",
                                            "len": 24
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:7f00::",
                                            "len": 24
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:a9fe::",
                                            "len": 32
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:ac10::",
                                            "len": 28
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:c0a8::",
                                            "len": 32
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:e000::",
                                            "len": 19
                                        }
                                    }
                                ]
                            }
                        }
                    },
                    {
                        "reject": {
                            "type": "icmpv6",
                            "expr": "addr-unreachable"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FORWARD",
                "handle": 180,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_FORWARD_ZONES"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FORWARD",
                "handle": 182,
                "expr": [
                    {
                        "reject": {
                            "type": "icmpx",
                            "expr": "admin-prohibited"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_OUTPUT",
                "handle": 184,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "ct": {
                                    "key": "state"
                                }
                            },
                            "right": {
                                "set": [
                                    "established",
                                    "related"
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_OUTPUT",
                "handle": 185,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "oifname"
                                }
                            },
                            "right": "lo"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_OUTPUT",
                "handle": 194,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "ip6",
                                    "field": "daddr"
                                }
                            },
                            "right": {
                                "set": [
                                    {
                                        "prefix": {
                                            "addr": "::",
                                            "len": 96
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "::ffff:0.0.0.0",
                                            "len": 96
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002::",
                                            "len": 24
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:a00::",
                                            "len": 24
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:7f00::",
                                            "len": 24
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:a9fe::",
                                            "len": 32
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:ac10::",
                                            "len": 28
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:c0a8::",
                                            "len": 32
                                        }
                                    },
                                    {
                                        "prefix": {
                                            "addr": "2002:e000::",
                                            "len": 19
                                        }
                                    }
                                ]
                            }
                        }
                    },
                    {
                        "reject": {
                            "type": "icmpv6",
                            "expr": "addr-unreachable"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_OUTPUT",
                "handle": 187,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_OUTPUT_POLICIES_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_OUTPUT",
                "handle": 189,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_OUTPUT_POLICIES_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_INPUT_POLICIES_pre",
                "handle": 307,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_policy_allow-host-ipv6"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_INPUT_ZONES",
                "handle": 314,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "eth0"
                        }
                    },
                    {
                        "goto": {
                            "target": "filter_IN_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_INPUT_ZONES",
                "handle": 269,
                "expr": [
                    {
                        "goto": {
                            "target": "filter_IN_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FORWARD_ZONES",
                "handle": 316,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "eth0"
                        }
                    },
                    {
                        "goto": {
                            "target": "filter_FWD_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FORWARD_ZONES",
                "handle": 271,
                "expr": [
                    {
                        "goto": {
                            "target": "filter_FWD_public"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public",
                "handle": 203,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_INPUT_POLICIES_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public",
                "handle": 204,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_public_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public",
                "handle": 205,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_public_log"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public",
                "handle": 206,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_public_deny"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public",
                "handle": 207,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_public_allow"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public",
                "handle": 208,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_public_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public",
                "handle": 209,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_INPUT_POLICIES_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public",
                "handle": 268,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "l4proto"
                                }
                            },
                            "right": {
                                "set": [
                                    "icmp",
                                    "ipv6-icmp"
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public",
                "handle": 210,
                "expr": [
                    {
                        "reject": {
                            "type": "icmpx",
                            "expr": "admin-prohibited"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public_allow",
                "handle": 211,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "tcp",
                                    "field": "dport"
                                }
                            },
                            "right": 22
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public_allow",
                "handle": 212,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "ip6",
                                    "field": "daddr"
                                }
                            },
                            "right": {
                                "prefix": {
                                    "addr": "fe80::",
                                    "len": 64
                                }
                            }
                        }
                    },
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "udp",
                                    "field": "dport"
                                }
                            },
                            "right": 546
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_public_allow",
                "handle": 213,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "tcp",
                                    "field": "dport"
                                }
                            },
                            "right": 9090
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POST_public",
                "handle": 220,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_POSTROUTING_POLICIES_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POST_public",
                "handle": 221,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_POST_public_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POST_public",
                "handle": 222,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_POST_public_log"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POST_public",
                "handle": 223,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_POST_public_deny"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POST_public",
                "handle": 224,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_POST_public_allow"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POST_public",
                "handle": 225,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_POST_public_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_POST_public",
                "handle": 226,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_POSTROUTING_POLICIES_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FWD_public",
                "handle": 233,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_FORWARD_POLICIES_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FWD_public",
                "handle": 234,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_FWD_public_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FWD_public",
                "handle": 235,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_FWD_public_log"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FWD_public",
                "handle": 236,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_FWD_public_deny"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FWD_public",
                "handle": 237,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_FWD_public_allow"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FWD_public",
                "handle": 238,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_FWD_public_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FWD_public",
                "handle": 239,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_FORWARD_POLICIES_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FWD_public",
                "handle": 240,
                "expr": [
                    {
                        "reject": {
                            "type": "icmpx",
                            "expr": "admin-prohibited"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_FWD_public_allow",
                "handle": 319,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "oifname"
                                }
                            },
                            "right": "eth0"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_public",
                "handle": 247,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PREROUTING_POLICIES_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_public",
                "handle": 248,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_public_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_public",
                "handle": 249,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_public_log"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_public",
                "handle": 250,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_public_deny"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_public",
                "handle": 251,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_public_allow"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_public",
                "handle": 252,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_public_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_public",
                "handle": 253,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PREROUTING_POLICIES_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_public",
                "handle": 260,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PREROUTING_POLICIES_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_public",
                "handle": 261,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_public_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_public",
                "handle": 262,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_public_log"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_public",
                "handle": 263,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_public_deny"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_public",
                "handle": 264,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_public_allow"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_public",
                "handle": 265,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_public_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_public",
                "handle": 266,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PREROUTING_POLICIES_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_policy_allow-host-ipv6",
                "handle": 280,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_policy_allow-host-ipv6_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_policy_allow-host-ipv6",
                "handle": 281,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_policy_allow-host-ipv6_log"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_policy_allow-host-ipv6",
                "handle": 282,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_policy_allow-host-ipv6_deny"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_policy_allow-host-ipv6",
                "handle": 283,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_policy_allow-host-ipv6_allow"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_policy_allow-host-ipv6",
                "handle": 284,
                "expr": [
                    {
                        "jump": {
                            "target": "filter_IN_policy_allow-host-ipv6_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_policy_allow-host-ipv6_allow",
                "handle": 310,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "icmpv6",
                                    "field": "type"
                                }
                            },
                            "right": "nd-neighbor-advert"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_policy_allow-host-ipv6_allow",
                "handle": 311,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "icmpv6",
                                    "field": "type"
                                }
                            },
                            "right": "nd-neighbor-solicit"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_policy_allow-host-ipv6_allow",
                "handle": 312,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "icmpv6",
                                    "field": "type"
                                }
                            },
                            "right": "nd-router-advert"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "filter_IN_policy_allow-host-ipv6_allow",
                "handle": 313,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "icmpv6",
                                    "field": "type"
                                }
                            },
                            "right": "nd-redirect"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_policy_allow-host-ipv6",
                "handle": 291,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_policy_allow-host-ipv6_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_policy_allow-host-ipv6",
                "handle": 292,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_policy_allow-host-ipv6_log"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_policy_allow-host-ipv6",
                "handle": 293,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_policy_allow-host-ipv6_deny"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_policy_allow-host-ipv6",
                "handle": 294,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_policy_allow-host-ipv6_allow"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "nat_PRE_policy_allow-host-ipv6",
                "handle": 295,
                "expr": [
                    {
                        "jump": {
                            "target": "nat_PRE_policy_allow-host-ipv6_post"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_policy_allow-host-ipv6",
                "handle": 302,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_policy_allow-host-ipv6_pre"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_policy_allow-host-ipv6",
                "handle": 303,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_policy_allow-host-ipv6_log"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_policy_allow-host-ipv6",
                "handle": 304,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_policy_allow-host-ipv6_deny"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_policy_allow-host-ipv6",
                "handle": 305,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_policy_allow-host-ipv6_allow"
                        }
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "firewalld",
                "chain": "mangle_PRE_policy_allow-host-ipv6",
                "handle": 306,
                "expr": [
                    {
                        "jump": {
                            "target": "mangle_PRE_policy_allow-host-ipv6_post"
                        }
                    }
                ]
            }
        }
    ]
}`
	if !gjson.Valid(jsonData) {
		fmt.Println("Invalid JSON data")
		return
	}

	wd := gjson.Get(jsonData, "nftables")
	tables := wd.Get("#.table").Array()
	tablesCount := len(tables)
	fmt.Println("Tables count: ", tablesCount)
	for _, table := range tables {
		fmt.Println("Table: ", table.Get("name").String())
	}
	//fmt.Printf("Tables: %#v\n", tables)
	chains := wd.Get("#.chain").Array()
	for _, chain := range chains {
		fmt.Println("Chain: ", chain.Get("name").String())
	}
	//fmt.Printf("Chains: %#v\n", chains)
	rules := wd.Get("#.rule").Array()
	for _, rule := range rules {
		fmt.Println("Rule: ", rule.Get("chain").String())
	}
	//fmt.Printf("Rules: %#v\n", rules)
}
