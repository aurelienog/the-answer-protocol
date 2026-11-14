// ==========================================================================
// CORE MODULE: AUDIO & MUSIC ENGINE (Sequential World Playlist + Combat BGM)
// ==========================================================================

const AudioManager = {
    muted: localStorage.getItem('tap_audio_muted') === 'true',
    unlocked: false,
    inCombatMusic: false,

    worldPlaylist: [
        '/assets/audio/music/arvendale_0.mp3',
        '/assets/audio/music/arvendale_1.mp3',
        '/assets/audio/music/arvendale_3.mp3',
        '/assets/audio/music/arvendale_4.mp3'
    ],
    combatTrackUrl: '/assets/audio/music/arvendale_combat.mp3',

    currentTrackIndex: 0,
    worldAudio: new Audio(),
    combatAudio: new Audio(),

    init() {
        this.worldAudio.volume = 0.35;
        this.combatAudio.volume = 0.42;
        this.combatAudio.loop = true;
        this.combatAudio.src = this.combatTrackUrl;

        // When one world track finishes, automatically play the next one in order!
        this.worldAudio.addEventListener('ended', () => {
            this.currentTrackIndex = (this.currentTrackIndex + 1) % this.worldPlaylist.length;
            this.playCurrentWorldTrack(true);
        });

        this.worldAudio.src = this.worldPlaylist[this.currentTrackIndex];
        this.syncMuteUI();

        // Unlock browser audio on first user interaction
        const unlock = () => {
            if (this.unlocked) return;
            this.unlocked = true;
            if (!this.muted) {
                this.resumeActiveMusic();
            }
            document.removeEventListener('click', unlock);
            document.removeEventListener('keydown', unlock);
        };
        document.addEventListener('click', unlock);
        document.addEventListener('keydown', unlock);
    },

    playCurrentWorldTrack(resetTime = false) {
        const targetSrc = this.worldPlaylist[this.currentTrackIndex];
        if (!this.worldAudio.src.endsWith(targetSrc)) {
            this.worldAudio.src = targetSrc;
        }
        if (resetTime) {
            this.worldAudio.currentTime = 0;
        }
        if (!this.muted && !this.inCombatMusic) {
            this.worldAudio.play().catch(() => {});
        }
    },

    nextTrack() {
        this.unlocked = true;
        // Switch to the next track in the world playlist
        this.currentTrackIndex = (this.currentTrackIndex + 1) % this.worldPlaylist.length;

        // If player is muted, unmute automatically when they explicitly click "Next Track"
        if (this.muted) {
            this.muted = false;
            localStorage.setItem('tap_audio_muted', 'false');
            this.syncMuteUI();
        }

        // If currently in combat, restart combat track; otherwise play the new world track!
        if (this.inCombatMusic) {
            this.combatAudio.currentTime = 0;
            this.combatAudio.play().catch(() => {});
        } else {
            this.playCurrentWorldTrack(true);
        }
    },

    enterCombatMusic() {
        if (this.inCombatMusic) return;
        this.inCombatMusic = true;
        this.worldAudio.pause();

        this.combatAudio.currentTime = 0;
        if (!this.muted) {
            this.combatAudio.play().catch(() => {});
        }
    },

    exitCombatMusic() {
        if (!this.inCombatMusic) return;
        this.inCombatMusic = false;
        this.combatAudio.pause();
        this.combatAudio.currentTime = 0;

        if (!this.muted) {
            this.worldAudio.play().catch(() => {});
        }
    },

    resumeActiveMusic() {
        if (this.inCombatMusic) {
            this.combatAudio.play().catch(() => {});
        } else {
            this.playCurrentWorldTrack(false);
        }
    },

    toggleMute() {
        this.muted = !this.muted;
        localStorage.setItem('tap_audio_muted', String(this.muted));
        this.syncMuteUI();

        if (this.muted) {
            this.worldAudio.pause();
            this.combatAudio.pause();
        } else {
            this.unlocked = true;
            this.resumeActiveMusic();
        }
    },

    syncMuteUI() {
        const iconPath = this.muted
            ? '/assets/icons/ui/sound_off.webp'
            : '/assets/icons/ui/sound_on.webp';
        const titleText = this.muted ? 'Unmute Audio' : 'Mute Audio';

        document.querySelectorAll('.js-sound-toggle').forEach(btn => {
            btn.title = titleText;
            btn.classList.toggle('sound-muted', this.muted);
            const img = btn.querySelector('img');
            if (img) img.src = iconPath;
        });
    },

    // Ready for your future Combat SFX (e.g. AudioManager.playSfx('hit'))!
    playSfx(name, volume = 0.6) {
        if (this.muted) return;
        const sfx = new Audio(`/assets/audio/sfx/${name}.mp3`);
        sfx.volume = volume;
        sfx.play().catch(() => {});
    }
};