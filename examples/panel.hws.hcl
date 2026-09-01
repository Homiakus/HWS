panel "main" {
  edge = "top"
  height = 40
  gap = 6
  overflow = "popover"

  group "applications" {
    source = "running"

    app {
      density = "adaptive"
      min_width = 64
      preferred_width = 156
      max_width = 240

      surfaces {
        mode = "segments"
        max_visible = 4
        overflow = "count"
      }
    }

    on "click" { action = "focus_or_cycle" }
    on "scroll" { action = "cycle_surface" }
    on "middle_click" { action = "new_window" }
    on "secondary_click" { action = "surface_menu" }
  }

  group "system" {
    widget "network" { variant = "mini" }
    widget "audio" { variant = "mini" }
    widget "clock" { format = "HH:mm" }
  }
}
